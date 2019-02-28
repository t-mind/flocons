#ifndef _FLOCONS_FILE_SERVICE_H_
#define _FLOCONS_FILE_SERVICE_H_

#include "../common.h"
#include "Directory.h"
#include "File.h"
#include "Path.h"
#include "RegularFile.h"

namespace flocons {

class FileService {
  public:
    virtual std::shared_ptr<File> getFile(const Path& path) = 0;
    std::shared_ptr<Directory> getDirectory(const Path& path);
    std::shared_ptr<RegularFile> getRegularFile(const Path& path);
    virtual std::shared_ptr<Directory> createDirectory(const Path& path, mode_t mode = 0755) = 0;
    virtual std::shared_ptr<RegularFile> createRegularFile(const Path& path, const char* data, size_t size, mode_t mode = 0644) = 0;
    virtual std::vector<std::shared_ptr<File>> listFiles(const Path& path) = 0;
};

} // namespace flocons

#endif // !_FLOCONS_FILE_SERVICE_H_