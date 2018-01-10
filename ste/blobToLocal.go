package main

import (
	"context"
	"github.com/Azure/azure-storage-blob-go/2016-05-31/azblob"
	"net/url"
	"time"
	"github.com/edsrzf/mmap-go"
	"fmt"
	"io"
	"sync/atomic"
	"github.com/Azure/azure-storage-azcopy/common"
)

type blobToLocal struct{
	// count the number of chunks that are done
	count uint32
}

func (blobToLocal blobToLocal) prologue(transfer TransferMsgDetail, chunkChannel chan<- ChunkMsg) {
	// step 1: get blob size
	p := azblob.NewPipeline(azblob.NewAnonymousCredential(), azblob.PipelineOptions{
		Retry: azblob.RetryOptions{
			Policy:        azblob.RetryPolicyExponential,
			MaxTries:      3,
			TryTimeout:    time.Second * 60,
			RetryDelay:    time.Second * 1,
			MaxRetryDelay: time.Second * 3,
		},
	})
	u, _ := url.Parse(transfer.Source)
	blobUrl := azblob.NewBlobURL(*u, p)
	blobSize := getBlobSize(blobUrl)

	// step 2: prep local file before download starts
	memoryMappedFile := createAndMemoryMapFile(transfer.Destination, blobSize)

	// step 3: go through the blob range and schedule download chunk jobs/msgs
	downloadChunkSize := int64(transfer.ChunkSize)
	blockIdCount := int32(0)
	for startIndex := int64(0); startIndex < blobSize; startIndex += downloadChunkSize {
		adjustedChunkSize := downloadChunkSize

		// compute exact size of the chunk
		if startIndex+downloadChunkSize > blobSize {
			adjustedChunkSize = blobSize - startIndex
		}

		// schedule the download chunk job
		chunkChannel <- ChunkMsg{
			doTransfer: generateDownloadFunc(
				transfer.JobId,
				transfer.PartNumber,
				transfer.TransferId,
				blockIdCount, // serves as index of chunk
				computeNumOfChunks(blobSize, downloadChunkSize),
				adjustedChunkSize,
				startIndex,
				blobUrl,
				memoryMappedFile,
				transfer.TransferCtx,
				transfer.TransferCancelFunc,
				&blobToLocal.count),
		}
		blockIdCount += 1
	}
}

// this generates a function which performs the downloading of a single chunk
func generateDownloadFunc(jobId common.JobID, partNum common.PartNumber,transferId uint32, chunkId int32, totalNumOfChunks uint32, chunkSize int64, startIndex int64,
	blobURL azblob.BlobURL, memoryMappedFile mmap.MMap, ctx context.Context, cancelTransfer func(), progressCount *uint32) chunkFunc {
	return func(workerId int) {
		transferIdentifierStr := fmt.Sprintf("jobId %d and partNum %d and transferId %d", jobId, partNum, transferId)

		fmt.Println("Worker", workerId, "is processing download CHUNK job with", transferIdentifierStr)

		// step 1: perform get
		get, err := blobURL.GetBlob(ctx, azblob.BlobRange{Offset: startIndex, Count: chunkSize}, azblob.BlobAccessConditions{}, false)
		if err != nil {
			// cancel entire transfer because this chunk has failed
			cancelTransfer()
			fmt.Println("Worker", workerId, "is canceling CHUNK job with", transferIdentifierStr, "and chunkID", chunkId, "because startIndex of", startIndex, "has failed")
			// TODO: update chunk status as failed
			return
		}

		// step2: write the body into the memory mapped file directly
		bytesRead, err := io.ReadFull(get.Body(), memoryMappedFile[startIndex: startIndex + chunkSize])
		get.Body().Close()
		if int64(bytesRead) != chunkSize || err != nil {
			// cancel entire transfer because this chunk has failed
			cancelTransfer()
			fmt.Println("Worker", workerId, "is canceling CHUNK job with", transferIdentifierStr, "and chunkID", chunkId, "because writing to file for startIndex of", startIndex, "has failed")
			// TODO: update chunk status as failed
			return
		}

		// TODO: update chunk status as success

		// step 3: check if this is the last chunk
		if atomic.AddUint32(progressCount, 1) == totalNumOfChunks {
			// step 4: this is the last block, perform EPILOGUE
			fmt.Println("Worker", workerId, "is concluding download TRANSFER job with", transferIdentifierStr, "after processing chunkId", chunkId)

			err := memoryMappedFile.Unmap()
			if err != nil {
				fmt.Println("Worker", workerId, "failed to conclude TRANSFER job with", transferIdentifierStr, "after processing chunkId", chunkId)
			}
		}
	}
}