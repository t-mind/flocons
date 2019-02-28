#ifndef _FILE_DIRECTORY_ACCESSOR_H_
#define _FILE_DIRECTORY_ACCESSOR_H_

#include "../../common.h"

namespace flocons {

class Path;
class Directory;
class File;
class RegularFile;
class DirectoryAccessor {
  public:
    virtual std::shared_ptr<File> getFile(const std::string& path) = 0;
    virtual std::shared_ptr<Directory> createDirectory(const std::string& path, mode_t mode) = 0;
    virtual std::shared_ptr<RegularFile> createRegularFile(const std::string& path, const char* data, size_t size, mode_t mode) = 0;
    virtual std::vector<std::shared_ptr<File>> listFiles() = 0;
};

} // namespace flocons

#include "../Directory.h"
#include "../File.h"
#include "../Path.h"
#include "../RegularFile.h"

#endif // !_FILE_DIRECTORY_ACCESSOR_H_