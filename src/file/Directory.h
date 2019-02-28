#ifndef _FLOCONS_FILE_DIRECTORY_H_
#define _FLOCONS_FILE_DIRECTORY_H_

#include "../common.h"
#include "File.h"

namespace flocons {

class DirectoryAccessor;
class RegularFile;
class Directory : public File {
  private:
    std::unique_ptr<DirectoryAccessor> _accessor;

  public:
    Directory(const Path& path, mode_t mode, time_t modification_time, std::unique_ptr<DirectoryAccessor>&& accessor)
        : File(File::Type::DIRECTORY, path, mode, modification_time, DIRECTORY_MIME_TYPE, 0), _accessor(std::move(accessor)) {}
    Directory(const Path& path, mode_t mode, time_t modification_time)
        : File(File::Type::DIRECTORY, path, mode, modification_time, DIRECTORY_MIME_TYPE, 0), _accessor(nullptr) {}

    void accessor(std::unique_ptr<DirectoryAccessor>&& accessor) { this->_accessor = std::move(accessor); }
    std::shared_ptr<File> getFile(const std::string& path);
    std::shared_ptr<Directory> createDirectory(const std::string& path, mode_t mode);
    std::shared_ptr<RegularFile> createRegularFile(const std::string& path, const char* data, size_t size, mode_t mode);
    std::vector<std::shared_ptr<File>> listFiles();
};

} // namespace flocons

#include "RegularFile.h"
#include "access/DirectoryAccessor.h"

#endif // !_FLOCONS_FILE_DIRECTORY_H_