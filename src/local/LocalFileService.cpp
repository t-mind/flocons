#include "LocalFileService.h"
#include "../Logger.h"
#include "../file/cache/DirectoryCache.h"
#include "LocalDirectoryAccessor.h"

namespace flocons {

LocalFileService::LocalFileService(const std::string& name, const Path& path) {
    Logger::debug << "New LocalFileService on " << path << std::endl;
    this->_context = std::make_shared<LocalContext>(name, path);
    this->_root_directory = std::make_shared<Directory>("/", 0755, time(NULL), std::make_unique<LocalDirectoryAccessor>(this->_context, "/"));
}

std::shared_ptr<File> LocalFileService::getFile(const Path& path) {
    std::shared_ptr<Directory> directory = this->_context->getDirectoryFromCache(path);
    if (directory != nullptr) return directory;

    directory = this->_context->getDirectoryFromCache(path.dirname());
    if (directory != nullptr) return directory->getFile(path.basename());

    directory = this->_root_directory;
    std::shared_ptr<File> file = directory;

    for (auto it = path.begin(); it != path.end(); it++) {
        if (directory == nullptr) throw std::system_error(ENOENT, std::system_category(), "RegularFile " + path + " does not exist");

        file = directory->getFile(*it);
        directory = std::dynamic_pointer_cast<Directory>(file);
    }
    return file;
}

std::shared_ptr<Directory> LocalFileService::createDirectory(const Path& path, mode_t mode) {
    return this->getDirectory(path.dirname())->createDirectory(path.basename(), mode);
}

std::shared_ptr<RegularFile> LocalFileService::createRegularFile(const Path& path, const char* data, size_t size, mode_t mode) {
    return this->getDirectory(path.dirname())->createRegularFile(path.basename(), data, size, mode);
}

std::vector<std::shared_ptr<File>> LocalFileService::listFiles(const Path& path) { return this->getDirectory(path)->listFiles(); }

} // namespace flocons