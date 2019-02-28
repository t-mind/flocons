#ifndef _FLOCONS_FILE_REGULAR_FILE_H_
#define _FLOCONS_FILE_REGULAR_FILE_H_

#include "../common.h"
#include "File.h"
#include "access/RegularFileAccessor.h"

namespace flocons {

class RegularFile : public File {
  private:
    size_t _size;
    std::unique_ptr<RegularFileAccessor> _accessor;

  public:
    RegularFile(const Path& path, size_t size, mode_t mode, time_t modification_time, size_t address, std::unique_ptr<RegularFileAccessor>&& accessor)
        : File(File::Type::REGULAR_FILE, path, mode, modification_time, RegularFile::getMimeType(path), address), _size(size), _accessor(std::move(accessor)) {}
    RegularFile(const Path& path, size_t size, mode_t mode, time_t modification_time, size_t address)
        : File(File::Type::REGULAR_FILE, path, mode, modification_time, RegularFile::getMimeType(path), address), _size(size), _accessor(nullptr) {}
    RegularFile(const Path& path, size_t size, mode_t mode, time_t modification_time, std::unique_ptr<RegularFileAccessor>&& accessor)
        : File(File::Type::REGULAR_FILE, path, mode, modification_time, RegularFile::getMimeType(path), 0), _size(size), _accessor(std::move(accessor)) {}
    RegularFile(const Path& path, size_t size, mode_t mode, time_t modification_time)
        : File(File::Type::REGULAR_FILE, path, mode, modification_time, RegularFile::getMimeType(path), 0), _size(size), _accessor(nullptr) {}

    void accessor(std::unique_ptr<RegularFileAccessor>&& accessor) { this->_accessor = std::move(accessor); }
    size_t size() const { return this->_size; };
    char* data();

    static const std::string& getMimeType(const std::string& extension);
    static const std::string& getMimeType(const Path& path) { return RegularFile::getMimeType(path.extension()); }
};

} // namespace flocons

#endif // !_FLOCONS_FILE_REGULAR_FILE_H_