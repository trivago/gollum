// +build ignore

#ifndef __INCLUED_WRAPPER_H__
#define __INCLUED_WRAPPER_H__

#include <librdkafka/rdkafka.h>

// errorHook is used in a message opaque pointer to store information required
// for routing messages back.
typedef struct errorHook {
     int topic;
     int index;
} ErrorHook_t;

// NewErrorHook creates a new handle that can be attached to massages. This
// allows callbacks to the topic that sent the message. 
ErrorHook_t* NewErrorHook(int topic, int index);

// RegisterErrorWrapper registers the internal error handler to the given
// config. This allows routing back errors to go.
void RegisterErrorWrapper(rd_kafka_conf_t* config);

// CreateBatch creates a new native kafka batch of the given size.
rd_kafka_message_t* CreateBatch(int size);

// DestroyBatch clears up the batch and any attached error handlers
void DestroyBatch(rd_kafka_message_t* pBatch, int length);

// StoreBatchItem stores data in the given message batch slot.
void StoreBatchItem(rd_kafka_message_t* pBatch, int index, void* key, int keyLen, void* payload, int payloadLen, void* hook);

// NextError finds the next error in a message batch (that has been sent).
// If no error is found -1 is returned.
int NextError(rd_kafka_message_t* pBatch, int length, int offset);

// GetErr returns the error code for a given message in the given batch.
int GetErr(rd_kafka_message_t* pBatch, int index);

#endif