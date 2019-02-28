#ifndef _REMOTE_HTTP_FILE_SERVICE_H_
#define _REMOTE_HTTP_FILE_SERVICE_H_

#include "../common.h"
#include "../file/FileService.h"
#include "../http/HTTPRequestPool.h"
#include "RemoteContext.h"

namespace flocons {

class HTTPFileService : public FileService {
  private:
    std::shared_ptr<RemoteContext> _context;
    HTTPRequestPool _request_pool;

    std::shared_ptr<Directory> _createDirectoryObject(const Path& path, mode_t mode, time_t modification_time);
    std::shared_ptr<RegularFile> _createRegularFileObject(const Path& path, const char* data, size_t size, mode_t mode, time_t modification_time);

  public:
    HTTPFileService(const std::string& host);
    virtual std::shared_ptr<File> getFile(const Path& path);
    virtual std::shared_ptr<Directory> createDirectory(const Path& path, mode_t mode = 0755);
    virtual std::shared_ptr<RegularFile> createRegularFile(const Path& path, const char* data, size_t size, mode_t mode = 0644);
    virtual std::vector<std::shared_ptr<File>> listFiles(const Path& path);
};

} // namespace flocons

#endif // !_REMOTE_HTTP_FILE_SERVICE_H_
