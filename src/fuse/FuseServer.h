#ifndef _FLOCONS_FUSE_SERVER_H_
#define _FLOCONS_FUSE_SERVER_H_

#include "../common.h"
#include "../file/FileService.h"

namespace flocons {

class FuseServer {
  private:
    std::shared_ptr<FileService> _file_service;

  public:
    FuseServer(const std::shared_ptr<FileService>& file_service) : _file_service(file_service) {}
    int run(int argc, char* argv[]);
    int run(const std::vector<std::string> args);

    // int do_getattr(const std::string& path, struct stat* st) { return 0; }
};

} // namespace flocons

#endif // !_FLOCONS_FUSE_SERVER_H_