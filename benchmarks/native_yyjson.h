#ifndef YYJSON_GO_BENCHMARKS_NATIVE_YYJSON_H
#define YYJSON_GO_BENCHMARKS_NATIVE_YYJSON_H

#include <stddef.h>
#include <stdint.h>
#include "../yyjson/src/yyjson.h"

yyjson_doc *bench_yyjson_read_copy(const char *data, size_t len);
size_t bench_yyjson_doc_value_count(yyjson_doc *doc);
yyjson_val *bench_yyjson_doc_root(yyjson_doc *doc);
yyjson_val *bench_yyjson_obj_getn(yyjson_val *obj, const char *key, size_t len);
size_t bench_yyjson_arr_size(yyjson_val *arr);
uint64_t bench_yyjson_get_uint(yyjson_val *val);
uint8_t bench_yyjson_get_bool(yyjson_val *val);
char *bench_yyjson_write_doc(const yyjson_doc *doc, size_t *len);
void bench_yyjson_doc_free(yyjson_doc *doc);
void bench_yyjson_free(void *ptr);

#endif
