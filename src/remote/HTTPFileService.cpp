#include "HTTPFileService.h"
#include "../Logger.h"
#include "../file/access/DataContainerRegularFileAccessor.h"
#include "../file/access/ProxyDirectoryAccessor.h"

namespace flocons {

HTTPFileService::HTTPFileService(const std::string& host) : _context(std::make_shared<RemoteContext>(host)) {}

std::shared_ptr<Directory> HTTPFileService::_createDirectoryObject(const Path& path, mode_t mode, time_t modification_time) {
    return std::make_shared<Directory>(
        path, mode, modification_time,
        std::make_unique<ProxyDirectoryAccessor>(
            [this, path](const std::string& relative_path) -> std::shared_ptr<File> { return this->getFile(path / relative_path); },
            [this, path](const std::string& relative_path, mode_t mode) -> std::shared_ptr<Directory> {
                return this->createDirectory(path / relative_path, mode);
            },
            [this, path](const std::string& relative_path, const char* data, size_t size, mode_t mode) -> std::shared_ptr<RegularFile> {
                return this->createRegularFile(path / relative_path, data, size, mode);
            },
            [this, path]() -> std::vector<std::shared_ptr<File>> { return this->listFiles(path); }));
}

std::shared_ptr<RegularFile> HTTPFileService::_createRegularFileObject(const Path& path, const char* data, size_t size, mode_t mode, time_t modification_time) {
    return std::make_shared<RegularFile>(path, size, mode, modification_time, std::make_unique<DataContainerRegularFileAccessor>(data, size));
}

std::shared_ptr<File> HTTPFileService::getFile(const Path& path) {
    HTTPCode return_code;
    auto host = this->_context->host();
    auto url = path.string();
    Logger::debug << "Get file " << url << " from host " << host << std::endl;

    do {
        auto request = this->_request_pool.getAndLockRequest(host);
        std::lock_guard<std::mutex> lock(request->mutex(), std::adopt_lock); // Be sure to unlock when out of function
        return_code = request->get(url);
        if (return_code == HTTPCode::OK) {
            if (request->mimeType() == "inode/directory") {
                return this->_createDirectoryObject(path, request->mode(), request->modification_time());
            } else
                return this->_createRegularFileObject(path, request->data(), request->dataSize(), request->mode(), request->modification_time());
        }
    } while (return_code == HTTPCode::TEMPORARY_REDIRECT);

    if (return_code == HTTPCode::NOT_FOUND)
        throw std::system_error(ENOENT, std::system_category(), "File " + path + " not found on server " + this->_context->host());
    throw std::system_error(1, std::system_category(),
                            "Unmanaged return code " + std::to_string(return_code) + " for file " + path + " not found on server " + this->_context->host());
}

std::shared_ptr<Directory> HTTPFileService::createDirectory(const Path& path, mode_t mode) {
    HTTPCode return_code;
    auto host = this->_context->host();
    auto url = path.string();
    Logger::debug << "Create directory " << url << " onto host " << host << std::endl;

    do {
        auto request = this->_request_pool.getAndLockRequest(host);
        std::lock_guard<std::mutex> lock(request->mutex(), std::adopt_lock); // Be sure to unlock when out of function
        return_code = request->put(url, NULL, 0, "inode/directory");
        if (return_code == HTTPCode::OK) return this->_createDirectoryObject(path, mode, time(NULL));
    } while (return_code == HTTPCode::TEMPORARY_REDIRECT);

    if (return_code == HTTPCode::NOT_FOUND)
        throw std::system_error(ENOENT, std::system_category(), "Directory " + path + " not found on server " + this->_context->host());
    throw std::system_error(1, std::system_category(),
                            "Unmanaged return code " + std::to_string(return_code) + " for directory " + path + " on server " + this->_context->host());
}

std::shared_ptr<RegularFile> HTTPFileService::createRegularFile(const Path& path, const char* data, size_t size, mode_t mode) {
    HTTPCode return_code;
    auto host = this->_context->host();
    auto url = path.string();
    Logger::debug << "Create file " << url << " onto host " << host << std::endl;

    do {
        auto request = this->_request_pool.getAndLockRequest(host);
        std::lock_guard<std::mutex> lock(request->mutex(), std::adopt_lock); // Be sure to unlock when out of function
        return_code = request->put(url, data, size);
        if (return_code == HTTPCode::OK) return this->_createRegularFileObject(path, data, size, mode, time(NULL));
    } while (return_code == HTTPCode::TEMPORARY_REDIRECT);

    if (return_code == HTTPCode::NOT_FOUND)
        throw std::system_error(ENOENT, std::system_category(), "Directory " + path.dirname() + " not found on server " + this->_context->host());
    throw std::system_error(1, std::system_category(),
                            "Unmanaged return code " + std::to_string(return_code) + " for file " + path + " on server " + this->_context->host());
}

std::vector<std::shared_ptr<File>> HTTPFileService::listFiles(const Path& path) { return std::vector<std::shared_ptr<File>>(); }

} // namespace flocons