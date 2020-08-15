#ifndef GO_CGO_ATRI_DEF_H
#define GO_CGO_ATRI_DEF_H
#include <stddef.h>
#include <stdint.h>
#include <stdlib.h>

typedef void (*ByteCallback)(uintptr_t, void*, void*, size_t); // ctx, result, error, length
void InvokeByteCallback(ByteCallback, uintptr_t, void*, void*, size_t);

#endif
