package filesys

import (
	"context"

	"github.com/chrislusf/seaweedfs/weed/filer2"
	"github.com/chrislusf/seaweedfs/weed/glog"
	"github.com/chrislusf/seaweedfs/weed/operation"
	"github.com/chrislusf/seaweedfs/weed/pb/filer_pb"
	"google.golang.org/grpc"
)

func (wfs *WFS) deleteFileChunks(chunks []*filer_pb.FileChunk) {
	if len(chunks) == 0 {
		return
	}

	var fileIds []string
	for _, chunk := range chunks {
		fileIds = append(fileIds, chunk.GetFileIdString())
	}

	wfs.WithFilerClient(func(client filer_pb.SeaweedFilerClient) error {
		deleteFileIds(wfs.option.GrpcDialOption, client, fileIds)
		return nil
	})
}

func deleteFileIds(grpcDialOption grpc.DialOption, client filer_pb.SeaweedFilerClient, fileIds []string) error {

	var vids []string
	for _, fileId := range fileIds {
		vids = append(vids, filer2.VolumeId(fileId))
	}

	lookupFunc := func(vids []string) (map[string]operation.LookupResult, error) {

		m := make(map[string]operation.LookupResult)

		glog.V(4).Infof("remove file lookup volume id locations: %v", vids)
		resp, err := client.LookupVolume(context.Background(), &filer_pb.LookupVolumeRequest{
			VolumeIds: vids,
		})
		if err != nil {
			return m, err
		}

		for _, vid := range vids {
			lr := operation.LookupResult{
				VolumeId:  vid,
				Locations: nil,
			}
			locations, found := resp.LocationsMap[vid]
			if !found {
				continue
			}
			for _, loc := range locations.Locations {
				lr.Locations = append(lr.Locations, operation.Location{
					Url:       loc.Url,
					PublicUrl: loc.PublicUrl,
				})
			}
			m[vid] = lr
		}

		return m, err
	}

	_, err := operation.DeleteFilesWithLookupVolumeId(grpcDialOption, fileIds, lookupFunc)

	return err
}
