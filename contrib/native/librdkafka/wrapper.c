// Copyright 2015-2016 trivago GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

void RegisterRandomPartitioner(rd_kafka_topic_conf_t* config) {
    rd_kafka_topic_conf_set_partitioner_cb(config, &rd_kafka_msg_partitioner_random);
}
                     
static int32_t msg_partitioner_round_robin(const rd_kafka_topic_t *rkt, const void *key, size_t keylen, int32_t partition_cnt, void *opaque, void *msg_opaque) {
    int32_t p = 0;
    int32_t tries = 0;
    do {
        int32_t index = __sync_fetch_and_add((int32_t*)opaque, 1);
        p = index % partition_cnt; 
        ++tries;
    } while (!rd_kafka_topic_partition_available(rkt, p) && tries < partition_cnt);
    return p;
}

void RegisterRoundRobinPartitioner(rd_kafka_topic_conf_t* config) {
    rd_kafka_topic_conf_set_opaque(config, calloc(1, sizeof(int32_t)));
    rd_kafka_topic_conf_set_partitioner_cb(config, &msg_partitioner_round_robin);
}

rd_kafka_message_t* CreateBatch(int size) {
    return (rd_kafka_message_t*)malloc(size * sizeof(rd_kafka_message_t));
}

void StoreBatchItem(rd_kafka_message_t* pBatch, int index, void* key, int keyLen, void* payload, int payloadLen, int topicId) {
     pBatch[index].key_len = (size_t)keyLen;
     pBatch[index].len = (size_t)payloadLen;
     pBatch[index].key = key;
     pBatch[index].payload = payload;
     pBatch[index]._private = NewErrorHook(topicId, index);
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