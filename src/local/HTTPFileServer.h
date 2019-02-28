#ifndef _FLOCONS_LOCAL_HTTP_FILE_SERVER_H
#define _FLOCONS_LOCAL_HTTP_FILE_SERVER_H

#include "../common.h"
#include "../file/FileService.h"

namespace flocons {

class HTTPFileServer {
  private:
    std::shared_ptr<FileService> _file_service;
    uint16_t _port;
    void* _daemon = NULL;

  public:
    HTTPFileServer(const std::shared_ptr<FileService>& file_service, uint16_t port) : _file_service(file_service), _port(port) {}
    ~HTTPFileServer() { this->stop(); }
    void start();
    void stop();
};

} // namespace flocons

#endif // !_FLOCONS_LOCAL_HTTP_FILE_SERVER_H