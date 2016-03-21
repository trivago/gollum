#include "wrapper.h"
#include <string.h>
#include <stdlib.h>

extern void goMarshalAsyncError(int, char*, void*);

ErrorHook_t* NewErrorHook(int topic, int index) {
     ErrorHook_t* hook = (ErrorHook_t*)malloc(sizeof(ErrorHook_t));
     hook->topic = topic;
     hook->index = index;
     return hook;
}

static void errorWrapper(rd_kafka_t* client, int code, const char* reason, void* opaque) {
     goMarshalAsyncError(code, (char*)reason, (ErrorHook_t*)opaque);
}

void RegisterErrorWrapper(rd_kafka_conf_t* config) {
    rd_kafka_conf_set_error_cb(config, &errorWrapper);
}

rd_kafka_message_t* CreateBatch(int size) {
     return (rd_kafka_message_t*)malloc(size * sizeof(rd_kafka_message_t));
}

void StoreBatchItem(rd_kafka_message_t* pBatch, int index, void* key, int keyLen, void* payload, int payloadLen, void* hook) {
     pBatch[index].key_len = (size_t)keyLen;
     pBatch[index].len = (size_t)payloadLen;
     pBatch[index].key = key;
     pBatch[index].payload = payload;
     pBatch[index]._private = hook;
}

void DestroyBatch(rd_kafka_message_t* pBatch, int length) {
    for (int i=0; i<length; ++i) {
        free(pBatch[i]._private);
    }
    free(pBatch);
}

int NextError(rd_kafka_message_t* pBatch, int length, int offset) {
    for (int i=offset; i<length; i++) {
        if (pBatch[i].err != RD_KAFKA_RESP_ERR_NO_ERROR) {
            return i;
        }
    }
    return -1;
}

int GetErr(rd_kafka_message_t* pBatch, int index) {
    return pBatch[index].err;
}