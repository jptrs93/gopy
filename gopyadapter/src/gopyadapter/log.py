"""Structured logging bridge from python to the parent go process.

Use it like the standard library logger::

    from gopyadapter import log

    log.info("starting work")
    log.warning("user %s missing field %s", user_id, field)
    log.error("failed", extra={"job_id": job_id})

    try:
        risky()
    except Exception:
        log.exception("risky failed", extra={"input": str(payload)})

Each call writes one base64-encoded msgpack line on stdout containing a
``{level, message, context}`` record. The parent go process attempts to decode
every stdout line; valid records are dispatched via slog at the matching
level, anything else falls through to the existing stdout handling. This
keeps the change fully backwards compatible — an older go side just logs the
encoded line verbatim.
"""

from __future__ import annotations

import base64
import logging
import sys
import traceback
from typing import Any, Optional

import msgpack

_LOGGER_NAME = "gopyadapter"

_LEVEL_NAMES = {
    logging.DEBUG: "DEBUG",
    logging.INFO: "INFO",
    logging.WARNING: "WARNING",
    logging.ERROR: "ERROR",
    logging.CRITICAL: "CRITICAL",
}

# LogRecord attributes set by the standard library. Anything else on the record
# was supplied via ``extra=`` and belongs in the structured context map.
_RESERVED_RECORD_ATTRS = frozenset(
    {
        "args",
        "asctime",
        "created",
        "exc_info",
        "exc_text",
        "filename",
        "funcName",
        "levelname",
        "levelno",
        "lineno",
        "message",
        "module",
        "msecs",
        "msg",
        "name",
        "pathname",
        "process",
        "processName",
        "relativeCreated",
        "stack_info",
        "taskName",
        "thread",
        "threadName",
    }
)


def _level_name(record: logging.LogRecord) -> str:
    return _LEVEL_NAMES.get(record.levelno, record.levelname or str(record.levelno))


def _build_context(record: logging.LogRecord) -> Optional[dict]:
    ctx: dict[str, str] = {}
    for k, v in record.__dict__.items():
        if k in _RESERVED_RECORD_ATTRS:
            continue
        ctx[k] = v if isinstance(v, str) else str(v)
    if record.exc_info:
        ctx["exception"] = "".join(traceback.format_exception(*record.exc_info)).rstrip()
    elif record.exc_text:
        ctx["exception"] = record.exc_text
    return ctx or None


class _GoStdoutHandler(logging.Handler):
    """Forwards python log records to the parent go process via stdout.

    The frame format is a single line of base64(msgpack(record)). Stdout is
    the same channel the go side already drains, so no protocol change is
    required and old/new combinations remain compatible.
    """

    def emit(self, record: logging.LogRecord) -> None:
        try:
            try:
                message = record.getMessage()
            except Exception:
                message = str(record.msg)
            payload = msgpack.packb(
                {
                    "level": _level_name(record),
                    "message": message,
                    "context": _build_context(record),
                },
                use_bin_type=True,
            )
            print(base64.b64encode(payload).decode("ascii"), flush=True)
        except Exception:
            # Mirrors logging's behaviour: never let logging crash the program.
            self.handleError(record)


_logger = logging.getLogger(_LOGGER_NAME)
_logger.setLevel(logging.DEBUG)
_logger.propagate = False

# Attach the handler eagerly: stdout is always available, so unlike the old
# fd-4-based bridge there is no need to defer until execute() runs.
if not any(isinstance(h, _GoStdoutHandler) for h in _logger.handlers):
    _handler = _GoStdoutHandler()
    _handler.setLevel(logging.DEBUG)
    _logger.addHandler(_handler)


def get_logger(name: Optional[str] = None) -> logging.Logger:
    """Return a logger that forwards to the parent go process.

    Without a name, returns the package logger. With a name, returns a child
    logger that inherits the bridge handler.
    """
    if name is None or name == _LOGGER_NAME:
        return _logger
    return _logger.getChild(name)


def debug(msg: Any, *args: Any, **kwargs: Any) -> None:
    _logger.debug(msg, *args, **kwargs)


def info(msg: Any, *args: Any, **kwargs: Any) -> None:
    _logger.info(msg, *args, **kwargs)


def warning(msg: Any, *args: Any, **kwargs: Any) -> None:
    _logger.warning(msg, *args, **kwargs)


def error(msg: Any, *args: Any, **kwargs: Any) -> None:
    _logger.error(msg, *args, **kwargs)


def critical(msg: Any, *args: Any, **kwargs: Any) -> None:
    _logger.critical(msg, *args, **kwargs)


def exception(msg: Any, *args: Any, **kwargs: Any) -> None:
    kwargs.setdefault("exc_info", sys.exc_info())
    _logger.error(msg, *args, **kwargs)


__all__ = [
    "debug",
    "info",
    "warning",
    "error",
    "critical",
    "exception",
    "get_logger",
]
