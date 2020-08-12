#include "def.h"

void InvokeByteCallback(ByteCallback cb, uintptr_t ctx, void* result, void* error, size_t length) {
    cb(ctx, result, error, length);
}
