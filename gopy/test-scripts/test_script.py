import numpy as np

from gopyadapter import log
from gopyadapter.core import execute


def add(i):
    print(i, flush=True)
    a = i['a']
    b = i['b']
    return {'result': a + b}


def add_scalar_output(i):
    print(i, flush=True)
    a = i['a']
    b = i['b']
    return a+b


def add_numpy_arrays(i):
    print(i, flush=True)
    a = i['a']
    b = i['b']
    return a+b


def identity(i):
    return i


def verify_2d_array(i):
    arr = i['arr2D']
    expected = np.array([[1.2,3.2],[99.1,-14.1]])
    if not np.array_equiv(arr, expected):
        raise Exception(f"expected arr {expected} but was {arr}")
    return i


def verify_1d_array(i):
    # print(f'{i}', flush=True)
    arr = i['arr1D']
    expected = np.array([1.2,3.2, 99.1,-14.1])
    if not np.array_equiv(arr, expected):
        raise Exception(f"expected arr {expected} but was {arr}")
    return i

def verify_1d_int32_array(i):
    arr = i['a']
    expected = np.array([10])
    if not np.array_equiv(arr, expected):
        raise Exception(f"expected arr {expected} but was {arr}")
    return i


def raises_value_error(i):
    raise ValueError(f"bad input: {i}")


def emit_logs(i):
    log.debug("debug message")
    log.info("hello %s", i.get("name", "world"))
    log.warning("missing field", extra={"field": "email"})
    try:
        raise RuntimeError("boom")
    except RuntimeError:
        log.exception("caught failure", extra={"job": "j1"})
    return {"emitted": 4}


if __name__ == '__main__':
    execute(**globals())
