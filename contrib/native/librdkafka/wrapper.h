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

// CreateBatch creates a new native kafka batch of the given size.
rd_kafka_message_t* CreateBatch(int size);

// DestroyBatch clears up the batch (without messages)
void DestroyBatch(rd_kafka_message_t* pBatch);

// StoreBatchItem stores data in the given message batch slot.
void StoreBatchItem(rd_kafka_message_t* pBatch, int index, size_t keyLen, void* pKey, size_t payloadLen, void* pPayload, size_t userdataLen, void* pUserdata);

// GetNextError finds the next error in a message batch (that has been sent).
// If no error is found -1 is returned.
int BatchGetNextError(rd_kafka_message_t* pBatch, int length, int offset);

// GetErr returns the error code for a given message in the given batch.
int BatchGetErrAt(rd_kafka_message_t* pBatch, int index);

// BatchGetBufferAt returns the userdata for a given message in the given batch.
buffer_t* BatchGetUserdataAt(rd_kafka_message_t* pBatch, int index);

// GetLastError returns rd_kafka_errno2err(errno)
int GetLastError();

// GetAllocatedBuffers returns the current allocation count for buffer_t
int64_t GetAllocatedBuffers();

#endif