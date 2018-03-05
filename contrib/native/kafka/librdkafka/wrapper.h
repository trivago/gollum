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

// +build ignore
#ifndef __INCLUED_WRAPPER_H__
#define __INCLUED_WRAPPER_H__

#include <librdkafka/rdkafka.h>

typedef struct buffer_s {
     void* data;
     int len;
} buffer_t;

// RegisterErrorWrapper registers the internal error handler to the given
// config. This allows routing back errors to go.
void RegisterErrorWrapper(rd_kafka_conf_t* config);

// RegisterDeliveryReportWrapper registers the internal handler for the
// successfull or unsuccessfull delivery of messages.
void RegisterDeliveryReportWrapper(rd_kafka_conf_t* config);

// RegisterRandomPartitioner registers the random partitioner to the given
// topic configuration.
void RegisterRandomPartitioner(rd_kafka_topic_conf_t* config);

// RegisterRoundRobinPartitioner registers a round robin partitioner to the
// given topic configuration.
void RegisterRoundRobinPartitioner(rd_kafka_topic_conf_t* config);

// CreateBuffer creates a buffer used to attach to message userdata
buffer_t* CreateBuffer(size_t len, void* pData);
 
// DestroyBuffer properly frees a buffer attached to message userdata
void DestroyBuffer(buffer_t* pBuffer);

// Alloc allocates a buffer of the given size and tracks the allocation
void* Alloc(size_t size);

// Free is the counterpart to Alloc and untracks an allocation
void Free(void* pData);

// Produce sends a message to a given topic
int Produce(rd_kafka_topic_t* pTopic, void* pKey, size_t keyLen, void* pPayload, size_t payloadLen, buffer_t* pUserdata);

// GetLastError returns rd_kafka_errno2err(errno)
int GetLastError();

// GetAllocCounter returns the number of currently open Alloc calls.
int64_t GetAllocCounter();

#endif