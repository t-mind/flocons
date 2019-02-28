#include "LocalDirectoryAccessor.h"
#include "../Logger.h"
#include "../file/cache/DirectoryCache.h"
#include "LocalContext.h"
#include "RegularFileContainer.h"

#include <dirent.h>
#include <sys/stat.h>
#include <unistd.h>

namespace flocons {

std::shared_ptr<Directory> LocalDirectoryAccessor::_createDirectoryObject(const Path& path, mode_t mode, time_t modification_time) {
    auto context = this->_context.lock();
    return std::make_shared<Directory>(path, mode, modification_time, std::make_unique<LocalDirectoryAccessor>(context, path));
}

std::shared_ptr<File> LocalDirectoryAccessor::getFile(const std::string& path) {
    auto context = this->_context.lock();
    auto full_path = this->_path / path;
    auto existing = context->getDirectoryFromCache(full_path);
    if (existing != nullptr) return existing;

    auto path_on_disk = context->path() / full_path;

    struct stat st;
    if (stat(path_on_disk.c_str(), &st) == -1) {
        if (errno == ENOENT) {
            auto file = this->getRegularFile(path);
            if (file != nullptr) return file;
        }
        throw std::system_error(errno, std::system_category(), "Error while looking for " + full_path);
    }

    if (S_ISDIR(st.st_mode)) {
        std::lock_guard<std::mutex> lock(this->_directory_cache_lock);
        existing = context->getDirectoryFromCache(full_path); // Could be created in the meantime by another thread
        if (existing != nullptr) return existing;

        auto directory = this->_createDirectoryObject(full_path, st.st_mode, st.st_mtime);
        context->addDirectoryToCache(full_path, directory);
        return directory;
    }

    throw std::system_error(ENOENT, std::system_category(), "Error while looking for " + full_path);
}

std::shared_ptr<Directory> LocalDirectoryAccessor::createDirectory(const std::string& path, mode_t mode) {
    auto context = this->_context.lock();
    auto full_path = this->_path / path;

    std::lock_guard<std::mutex> lock(this->_directory_cache_lock);
    auto existing = context->getDirectoryFromCache(full_path);
    if (existing != nullptr) throw std::system_error(EEXIST, std::system_category(), "Error while creating " + full_path);

    auto path_on_disk = context->path() / full_path;
    if (mkdir(path_on_disk.c_str(), mode) == -1) throw std::system_error(errno, std::system_category(), "Error while creating " + full_path);

    auto directory = this->_createDirectoryObject(full_path, mode, time(NULL));
    context->addDirectoryToCache(full_path, directory);
    return directory;
}

std::shared_ptr<RegularFile> LocalDirectoryAccessor::getRegularFile(const std::string& path) {
    std::shared_ptr<RegularFile> file = nullptr;
    for (auto it = this->_file_containers.begin(); file == nullptr && it != this->_file_containers.end(); ++it) {
        try {
            file = it->second->getRegularFile(path);
        } catch (std::system_error e) { Logger::error << e.what() << ": " << strerror(e.code().value()) << std::endl; }
    }
    if (file == nullptr) {
        auto added_containers = this->_refresh_file_containers();
        for (auto it = added_containers.begin(); file == nullptr && it != added_containers.end(); ++it) {
            try {
                file = (*it)->getRegularFile(path);
            } catch (std::system_error e) { Logger::error << e.what() << ": " << strerror(e.code().value()) << std::endl; }
        }
    }
    return file;
}

std::shared_ptr<RegularFile> LocalDirectoryAccessor::createRegularFile(const std::string& path, const char* data, size_t size, mode_t mode) {
    if (this->_current_writable_file_container == nullptr) {
        std::lock_guard<std::recursive_mutex> lock(this->_container_lock); // lock containers until end of method
        this->_refresh_file_containers();
        if (this->_file_containers.size() > 0) {
            for (auto it = this->_file_containers.begin(); it != this->_file_containers.end(); ++it) {
                if (it->second->mode() == LocalContext::FileMode::local &&
                    (this->_current_writable_file_container == nullptr || this->_current_writable_file_container->order() < it->second->order()))
                    this->_current_writable_file_container = it->second;
            }
        }
        if (this->_current_writable_file_container == nullptr) this->_new_writable_file_container();
    }
    return this->_current_writable_file_container->writeRegularFile(path, size, data, mode);
}

std::vector<std::shared_ptr<File>> LocalDirectoryAccessor::listFiles() {
    auto context = this->_context.lock();
    std::vector<std::shared_ptr<File>> files;
    this->_refresh_file_containers();

    DIR* d = opendir((context->path() / this->_path).c_str());
    if (d == NULL) throw std::system_error(errno, std::system_category(), "Error while browsing " + this->_path + " for file structure");
    struct dirent* result = NULL;
    struct dirent* entry = (struct dirent*)malloc(offsetof(struct dirent, d_name) + NAME_MAX + 1);
    if (entry == NULL)
        throw std::system_error(errno, std::system_category(), "Error while browsing " + this->_path + " for file structure, can't allocate memory");

// Ignore warning, readdir_r is deprecated because it does not support really long file name but we don't care here
// And it is usefull for multithreading
#pragma GCC diagnostic push
#pragma GCC diagnostic ignored "-Wdeprecated-declarations"
    while (readdir_r(d, entry, &result) == 0 && result != NULL) {
#pragma GCC diagnostic pop
        if (entry->d_type == DT_DIR) {
            auto full_path = this->_path / entry->d_name;
            auto directory = context->getDirectoryFromCache(full_path);
            if (directory == nullptr) {
                struct stat st;
                stat((context->path() / full_path).c_str(), &st);
                directory = this->_createDirectoryObject(full_path, st.st_mode, st.st_mtime);
            }
            files.push_back(directory);
        }
    }

    for (auto it = this->_file_containers.begin(); it != this->_file_containers.end(); ++it) {
        auto index_files = it->second->listRegularFiles();
        files.reserve(files.size() + index_files.size());
        std::move(index_files.begin(), index_files.end(), std::back_inserter(files));
    }
    closedir(d);
    free(entry);
    return files;
}

std::vector<std::shared_ptr<RegularFileContainer>> LocalDirectoryAccessor::_refresh_file_containers() {
    auto context = this->_context.lock();
    Logger::debug << "Refresh file containers for directory " + this->_path + " in context " + context->name() << std::endl;

    std::vector<std::shared_ptr<RegularFileContainer>> added_containers;
    DIR* d = opendir((context->path() / this->_path).c_str());
    if (d == NULL) throw std::system_error(errno, std::system_category(), "Error while browsing " + (context->path() / this->_path) + " for file structure");
    struct dirent* result = NULL;
    struct dirent* entry = (struct dirent*)malloc(offsetof(struct dirent, d_name) + NAME_MAX + 1);
    if (entry == NULL)
        throw std::system_error(errno, std::system_category(), "Error while browsing " + this->_path + " for file structure, can't allocate memory");

// Ignore warning, readdir_r is deprecated because it does not support really long file name but we don't care here
// And it is usefull for multithreading
#pragma GCC diagnostic push
#pragma GCC diagnostic ignored "-Wdeprecated-declarations"
    while (readdir_r(d, entry, &result) == 0 && result != NULL) {
#pragma GCC diagnostic pop
        if (entry->d_type == DT_REG) {
            static __thread char context_name[255];
            static __thread char extension[5];
            static __thread int version;
            static __thread int order;

            if (sscanf(entry->d_name, "files_%[^_]_v%d_%d.%s", context_name, &version, &order, extension) == 4) {
                Logger::debug << "Found entry " << entry->d_name << std::endl;
                std::string filename(entry->d_name, strlen(entry->d_name) - strlen(extension) - 1);

                std::lock_guard<std::recursive_mutex> lock(this->_container_lock); // lock containers until end of if
                if (this->_file_containers.find(filename) == this->_file_containers.end()) {
                    auto container = std::make_shared<RegularFileContainer>(context, filename, order, this->_path,
                                                                            context->name() == context_name ? LocalContext::FileMode::local
                                                                                                            : LocalContext::FileMode::remote);
                    this->_file_containers[filename] = container;
                    added_containers.push_back(container);
                }
            }
        }
    }
    closedir(d);
    free(entry);
    return added_containers;
}

void LocalDirectoryAccessor::_new_writable_file_container() {
    std::lock_guard<std::recursive_mutex> lock(this->_container_lock); // lock containers until end of method
    auto context = this->_context.lock();
    int order = 0;
    for (auto it = this->_file_containers.begin(); it != this->_file_containers.end(); ++it) {
        if (it->second->mode() == LocalContext::FileMode::local && it->second->order() > order) order = it->second->order();
    }
    order++;
    static __thread char filename[255];
    sprintf(filename, "files_%s_v%d_%d", context->name().c_str(), 0, order);
    Logger::debug << "Create new writable file container for folder " << this->_path << " with name " << filename << std::endl;
    this->_current_writable_file_container = std::make_shared<RegularFileContainer>(context, filename, order, this->_path, LocalContext::FileMode::local);
    this->_file_containers[filename] = this->_current_writable_file_container;
}

} // namespace flocons