#ifndef _FLOCONS_FILE_PROXY_DIRECTORY_ACCESSOR_H_
#define _FLOCONS_FILE_PROXY_DIRECTORY_ACCESSOR_H_

#include "../../common.h"
#include "DirectoryAccessor.h"

namespace flocons {

class ProxyDirectoryAccessor : public DirectoryAccessor {
  private:
    std::function<std::shared_ptr<File>(const std::string&)> _get_file;
    std::function<std::shared_ptr<Directory>(const std::string&, mode_t)> _create_directory;
    std::function<std::shared_ptr<RegularFile>(const std::string&, const char*, size_t, mode_t)> _create_regular_file;
    std::function<std::vector<std::shared_ptr<File>>()> _list_files;

  public:
    ProxyDirectoryAccessor(std::function<std::shared_ptr<File>(const std::string&)> get_file,
                           std::function<std::shared_ptr<Directory>(const std::string&, mode_t)> create_directory,
                           std::function<std::shared_ptr<RegularFile>(const std::string&, const char*, size_t, mode_t)> create_regular_file,
                           std::function<std::vector<std::shared_ptr<File>>()> list_files)
        : _get_file(get_file), _create_directory(create_directory), _create_regular_file(create_regular_file), _list_files(list_files) {}

    virtual std::shared_ptr<File> getFile(const std::string& path) {
        if (this->_get_file != nullptr) return this->_get_file(path);
        return nullptr;
    }
    virtual std::shared_ptr<Directory> createDirectory(const std::string& path, mode_t mode) {
        if (this->_create_directory != nullptr) return this->_create_directory(path, mode);
        return nullptr;
    }
    virtual std::shared_ptr<RegularFile> createRegularFile(const std::string& path, const char* data, size_t size, mode_t mode) {
        if (this->_create_regular_file != nullptr) return this->_create_regular_file(path, data, size, mode);
        return nullptr;
    }
    virtual std::vector<std::shared_ptr<File>> listFiles() {
        if (this->_list_files != nullptr) return this->_list_files();
        return std::vector<std::shared_ptr<File>>();
    }
};

} // namespace flocons

#include "../Directory.h"

#endif // !_FLOCONS_FILE_PROXY_DIRECTORY_ACCESSOR_H_