//
// Created by jfs on 11/01/23.
//

#include <grpcpp/grpcpp.h>

#include "./hub.pb.h"
#include "./hub.grpc.pb.h"

class MediaService :
        public grpc::Service,
        public cams::api::hub::Downstream::StubInterface {
public:
    MediaService() : grpc::Service(), cams::api::hub::Downstream::StubInterface() {};
    virtual ~MediaService() override {};

};

int main(int argc, char **argv) {
    int real_port{0};
    grpc::ServerBuilder builder;
    MediaService service;
    builder.SetResourceQuota(grpc::ResourceQuota().SetMaxThreads(1));
    builder.AddListeningPort("127.0.0.1:6001", grpc::InsecureServerCredentials(), &real_port);
    builder.SetDefaultCompressionAlgorithm(GRPC_COMPRESS_NONE);
    builder.SetDefaultCompressionLevel(GRPC_COMPRESS_LEVEL_NONE);
    builder.RegisterService()
    return 0;
}