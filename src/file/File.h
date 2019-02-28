#ifndef _FLOCONS_FILE_H_
#define _FLOCONS_FILE_H_

#include "../common.h"
#include "Path.h"

#define DIRECTORY_MIME_TYPE "inode/directory"
#define DEFAULT_FILE_MIME_TYPE "application/octet-stream"
#define JPEG_MIME_TYPE "image/jpeg"
#define MP4_MIME_TYPE "video/mp4"
#define TXT_MIME_TYPE "file/txt"

namespace flocons {

class File {
  public:
    enum class Type { REGULAR_FILE, DIRECTORY };

  private:
    Type _type;
    Path _path;
    mode_t _mode;
    time_t _modification_time;
    std::string _mime_type;
    size_t _address;

  public:
    File(Type type, const Path& path, mode_t mode, time_t modification_time, const std::string& mime_type, size_t address)
        : _type(type), _path(path), _mode(mode), _modification_time(modification_time), _mime_type(mime_type), _address(address) {}
    virtual ~File() {}

    Type type() const { return this->_type; }
    const std::string& name() const { return this->_path.basename(); }
    const std::string& extension() const { return this->_path.extension(); }
    const std::string& mimeType() const { return this->_mime_type; }
    const Path& path() const { return this->_path; }
    mode_t mode() const { return this->_mode; }
    time_t modificationTime() const { return this->_modification_time; }
    size_t address() const { return this->_address; }
};

} // namespace flocons

#endif // !_FLOCONS_FILE_H_