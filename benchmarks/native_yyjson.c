//go:build cgo

#include "native_yyjson.h"

#include <string.h>
#include <stdlib.h>

#include "../yyjson/src/yyjson.c"

yyjson_doc *bench_yyjson_read_copy(const char *data, size_t len) {
    if (len >= ((size_t)-1) - YYJSON_PADDING_SIZE) return NULL;

    size_t size = len + YYJSON_PADDING_SIZE;
    char *buf = malloc(size);
    if (!buf) return NULL;

    memcpy(buf, data, len);
    memset(buf + len, 0, YYJSON_PADDING_SIZE);

    yyjson_doc *doc = yyjson_read(buf, len, 0);
    free(buf);
    return doc;
}

size_t bench_yyjson_doc_value_count(yyjson_doc *doc) {
    return yyjson_doc_get_val_count(doc);
}

yyjson_val *bench_yyjson_doc_root(yyjson_doc *doc) {
    return yyjson_doc_get_root(doc);
}

yyjson_val *bench_yyjson_obj_getn(yyjson_val *obj, const char *key, size_t len) {
    return yyjson_obj_getn(obj, key, len);
}

size_t bench_yyjson_arr_size(yyjson_val *arr) {
    return yyjson_arr_size(arr);
}

uint64_t bench_yyjson_get_uint(yyjson_val *val) {
    return yyjson_get_uint(val);
}

uint8_t bench_yyjson_get_bool(yyjson_val *val) {
    return yyjson_get_bool(val);
}

char *bench_yyjson_write_doc(const yyjson_doc *doc, size_t *len) {
    return yyjson_write(doc, 0, len);
}

void bench_yyjson_doc_free(yyjson_doc *doc) {
    yyjson_doc_free(doc);
}

void bench_yyjson_free(void *ptr) {
    free(ptr);
}
