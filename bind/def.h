#ifndef GO_CGO_DEF_TEST_H
#define GO_CGO_DEF_TEST_H
#include <stddef.h>
#include <stdint.h>

typedef void (*ByteCallback)(uintptr_t, void*, void*, size_t); // ctx, result, error, length
void InvokeByteCallback(ByteCallback, uintptr_t, void*, void*, size_t);

#endif
