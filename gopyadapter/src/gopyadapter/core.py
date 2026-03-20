import msgpack
import os
import numpy as np
import struct
import traceback

# Extension codes for numpy array types and dimensions
EXT_FLOAT16 = 1
EXT_FLOAT16_2D = 2
EXT_FLOAT16_3D = 3

EXT_FLOAT32 = 11
EXT_FLOAT32_2D = 12
EXT_FLOAT32_3D = 13

EXT_FLOAT64 = 21
EXT_FLOAT64_2D = 22
EXT_FLOAT64_3D = 23

EXT_INT16 = 31
EXT_INT16_2D = 32
EXT_INT16_3D = 33

EXT_INT32 = 41
EXT_INT32_2D = 42
EXT_INT32_3D = 43

EXT_INT64 = 51
EXT_INT64_2D = 52
EXT_INT64_3D = 53

# Mapping from numpy types to extension codes and byte formats
NUMPY_TYPE_MAP = {
    np.float16: {
        1: (EXT_FLOAT16, '<f2'),
        2: (EXT_FLOAT16_2D, '<f2'),
        3: (EXT_FLOAT16_3D, '<f2')
    },
    np.float32: {
        1: (EXT_FLOAT32, '<f4'),
        2: (EXT_FLOAT32_2D, '<f4'),
        3: (EXT_FLOAT32_3D, '<f4')
    },
    np.float64: {
        1: (EXT_FLOAT64, '<f8'),
        2: (EXT_FLOAT64_2D, '<f8'),
        3: (EXT_FLOAT64_3D, '<f8')
    },
    np.int16: {
        1: (EXT_INT16, '<i2'),
        2: (EXT_INT16_2D, '<i2'),
        3: (EXT_INT16_3D, '<i2')
    },
    np.int32: {
        1: (EXT_INT32, '<i4'),
        2: (EXT_INT32_2D, '<i4'),
        3: (EXT_INT32_3D, '<i4')
    },
    np.int64: {
        1: (EXT_INT64, '<i8'),
        2: (EXT_INT64_2D, '<i8'),
        3: (EXT_INT64_3D, '<i8')
    }
}

# Reverse mapping from extension codes to numpy types and shapes
EXT_TYPE_MAP = {
    EXT_FLOAT16: (np.dtype('<f2'), 1),
    EXT_FLOAT16_2D: (np.dtype('<f2'), 2),
    EXT_FLOAT16_3D: (np.dtype('<f2'), 3),
    EXT_FLOAT32: (np.dtype('<f4'), 1),
    EXT_FLOAT32_2D: (np.dtype('<f4'), 2),
    EXT_FLOAT32_3D: (np.dtype('<f4'), 3),
    EXT_FLOAT64: (np.dtype('<f8'), 1),
    EXT_FLOAT64_2D: (np.dtype('<f8'), 2),
    EXT_FLOAT64_3D: (np.dtype('<f8'), 3),
    EXT_INT16: (np.dtype('<i2'), 1),
    EXT_INT16_2D: (np.dtype('<i2'), 2),
    EXT_INT16_3D: (np.dtype('<i2'), 3),
    EXT_INT32: (np.dtype('<i4'), 1),
    EXT_INT32_2D: (np.dtype('<i4'), 2),
    EXT_INT32_3D: (np.dtype('<i4'), 3),
    EXT_INT64: (np.dtype('<i8'), 1),
    EXT_INT64_2D: (np.dtype('<i8'), 2),
    EXT_INT64_3D: (np.dtype('<i8'), 3)
}

# MessagePack extension packer for numpy arrays
def default(obj):
    if isinstance(obj, np.ndarray):
        ndim = obj.ndim
        # Check if we support this data type and dimensionality
        if obj.dtype.type in NUMPY_TYPE_MAP and ndim in NUMPY_TYPE_MAP[obj.dtype.type]:
            ext_code, byte_format = NUMPY_TYPE_MAP[obj.dtype.type][ndim]
            byte_data = obj.astype(byte_format).tobytes()

            if ndim > 1:
                # Instead of metadata object, use 4-byte integers for each dimension
                shape_bytes = b''
                for dim_size in obj.shape:
                    shape_bytes += struct.pack('<I', dim_size)  # 4-byte little-endian unsigned int
                packed_data = shape_bytes + byte_data
            else:
                packed_data = byte_data

            return msgpack.ExtType(ext_code, packed_data)
        else:
            raise TypeError(f"Unsupported dtype: {obj.dtype} or dimension: {ndim}")
    raise TypeError(f'Object of type {obj.__class__.__name__} is not MessagePack serializable')

# MessagePack extension unpacker for numpy arrays
def ext_hook(code, data):
    if code in EXT_TYPE_MAP:
        dtype, ndim = EXT_TYPE_MAP[code]
        if ndim > 1:
            shape = np.frombuffer(data[:ndim*4],'<i4')
            binary_data = data[ndim*4:]
            array = np.frombuffer(binary_data, dtype=dtype)
            array = array.reshape(shape)
        else:
            array = np.frombuffer(data, dtype=dtype)
        return array

    return msgpack.ExtType(code, data)

def execute(**kwargs):
    rd, wd = 3, 4  # the read and write pipe indexes
    with os.fdopen(rd, "rb") as rf:
        os.write(wd, "ready".encode())
        while True:
            to_read = int.from_bytes(rf.read(4), "big")
            func_name_bytes = rf.read(to_read)
            func_name = func_name_bytes.decode()

            to_read = int.from_bytes(rf.read(4), "big")
            func_input_data = rf.read(to_read)
            func_input = msgpack.unpackb(func_input_data, ext_hook=ext_hook, raw=False)

            try:
                result = kwargs[func_name](func_input)
                msg_to_write = msgpack.packb(result, default=default, use_bin_type=True)
                x = int.to_bytes(len(msg_to_write), 4, "big", signed=True)
            except Exception as e:
                error_payload = {
                    "type": e.__class__.__name__,
                    "message": str(e),
                    "traceback": traceback.format_exc(),
                }
                msg_to_write = msgpack.packb(error_payload, use_bin_type=True)
                x = int.to_bytes(-len(msg_to_write), 4, "big", signed=True)

            os.write(wd, x)
            os.write(wd, msg_to_write)
