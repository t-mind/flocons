#include "FileService.h"

namespace flocons {

std::shared_ptr<Directory> FileService::getDirectory(const Path& path) {
    auto file = this->getFile(path);
    if (file == nullptr || file->type() != File::Type::DIRECTORY)
        throw std::system_error(ENOTDIR, std::system_category(), "RegularFile " + path + " is not a directory");
    return std::dynamic_pointer_cast<Directory>(file);
}

std::shared_ptr<RegularFile> FileService::getRegularFile(const Path& path) {
    auto file = this->getFile(path);
    if (file == nullptr || file->type() != File::Type::REGULAR_FILE)
        throw std::system_error(EISDIR, std::system_category(), "RegularFile " + path + " is not a regular file");
    return std::dynamic_pointer_cast<RegularFile>(file);
}

} // namespace flocons