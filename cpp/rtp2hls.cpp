//
// Created by jfs on 11/01/23.
//

#include <grpcpp/grpcpp.h>

#include "./hub.pb.h"
#include "./hub.grpc.pb.h"
#include "./MediaService.hpp"

int main([[maybe_unused]] int argc, [[maybe_unused]] char **argv) {
    int real_port{0};
    MediaService service;

    grpc::ServerBuilder builder;
    builder.SetResourceQuota(grpc::ResourceQuota().SetMaxThreads(2));
    builder.AddListeningPort("127.0.0.1:6001", grpc::InsecureServerCredentials(), &real_port);
    builder.SetDefaultCompressionAlgorithm(GRPC_COMPRESS_NONE);
    builder.SetDefaultCompressionLevel(GRPC_COMPRESS_LEVEL_NONE);
    builder.RegisterService(&service);

    auto server = builder.BuildAndStart();
    server->Wait();
    return 0;
}