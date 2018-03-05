// Copyright 2015-2018 trivago N.V.
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
#include <errno.h>


// ------------------------------------
// exported from go
// ------------------------------------

extern void goErrorHandler(int, char*);
extern void goLogHandler(int, char*, char*);
extern void goDeliveryHandler(rd_kafka_t*, int, buffer_t*);

int64_t gAllocCounter = 0;

// ------------------------------------
// static helper functions and wrapper
// ------------------------------------

static void errorWrapper(rd_kafka_t* client, int code, const char* reason, void* opaque) {
     goErrorHandler(code, (char*)reason);
}

static void logWrapper(const rd_kafka_t *rk, int level, const char *fac, const char *buf) {
    goLogHandler(level, (char*)fac, (char*)buf);
}

static void deliveryWrapper(rd_kafka_t* rk, const rd_kafka_message_t* rkmessage, void *opaque) {
    buffer_t* pBuffer = (buffer_t*)rkmessage->_private;
    goDeliveryHandler(rk, rkmessage->err, pBuffer);
    DestroyBuffer(pBuffer);
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

// ------------------------------------
// API helper
// ------------------------------------

void RegisterErrorWrapper(rd_kafka_conf_t* config) {
    rd_kafka_conf_set_error_cb(config, errorWrapper);
    rd_kafka_conf_set_log_cb(config, logWrapper);
}

void RegisterRandomPartitioner(rd_kafka_topic_conf_t* config) {
    rd_kafka_topic_conf_set_partitioner_cb(config, rd_kafka_msg_partitioner_random);
}

void RegisterDeliveryReportWrapper(rd_kafka_conf_t* config) {
    rd_kafka_conf_set_dr_msg_cb(config, deliveryWrapper);
}

void RegisterRoundRobinPartitioner(rd_kafka_topic_conf_t* config) {
    rd_kafka_topic_conf_set_opaque(config, calloc(1, sizeof(int32_t)));
    rd_kafka_topic_conf_set_partitioner_cb(config, msg_partitioner_round_robin);
}

buffer_t* CreateBuffer(size_t len, void* pData) {
    if (pData == NULL || len == 0) {
        return NULL;
    }
    
    buffer_t* pBuffer = (buffer_t*)Alloc(sizeof(buffer_t));
    pBuffer->data = pData;
    pBuffer->len = len;
    return pBuffer;
}

void DestroyBuffer(buffer_t* pBuffer) {
    if (pBuffer != NULL) {
        Free(pBuffer->data);
        Free(pBuffer);
    }
}

void* Alloc(size_t size) {
    __sync_fetch_and_add(&gAllocCounter, 1);
    return malloc(size);
}

void Free(void* pData) {
    if (pData != NULL) {
        free(pData);
        __sync_fetch_and_sub(&gAllocCounter, 1);
    }
}

int Produce(rd_kafka_topic_t* pTopic, void* pKey, size_t keyLen, void* pPayload, size_t payloadLen, buffer_t* pUserdata) {
    return rd_kafka_produce(pTopic, RD_KAFKA_PARTITION_UA, RD_KAFKA_MSG_F_COPY, pPayload, payloadLen, pKey, keyLen, pUserdata);
}
 
int GetLastError() {
    return rd_kafka_errno2err(errno);
}

int64_t GetAllocCounter() {
    return gAllocCounter;
}