#include "Directory.h"

namespace flocons {
std::shared_ptr<File> Directory::getFile(const std::string& path) {
    if (this->_accessor != nullptr) return this->_accessor->getFile(path);
    return nullptr;
}

std::shared_ptr<Directory> Directory::createDirectory(const std::string& path, mode_t mode) {
    if (this->_accessor != nullptr) return this->_accessor->createDirectory(path, mode);
    return nullptr;
}

std::shared_ptr<RegularFile> Directory::createRegularFile(const std::string& path, const char* data, size_t size, mode_t mode) {
    if (this->_accessor != nullptr) return this->_accessor->createRegularFile(path, data, size, mode);
    return nullptr;
}

std::vector<std::shared_ptr<File>> Directory::listFiles() {
    if (this->_accessor != nullptr) return this->_accessor->listFiles();
    return std::vector<std::shared_ptr<File>>();
}

} // namespace flocons