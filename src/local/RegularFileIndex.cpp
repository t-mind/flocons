#include "RegularFileIndex.h"
#include "../Logger.h"
#include "../file/serialize/CSVFileSerializer.h"
#include <sys/stat.h>
#include <unistd.h>

namespace flocons {
RegularFileIndex::RegularFileIndex(const std::shared_ptr<LocalContext>& context, const std::string& filename, LocalContext::FileMode mode, const Path& dir_path)
    : _context(context), _filename(filename), _mode(mode), _dir_path(dir_path) {
    _path_on_disk = context->path() / dir_path / (filename + ".csv");
    Logger::debug << "Create regular file index " << filename << " with mode " << (mode == LocalContext::FileMode::local ? "local" : "remote") << std::endl;
}

int RegularFileIndex::count() {
    this->refresh();
    return this->_cache.size();
}

std::shared_ptr<RegularFile> RegularFileIndex::get(const std::string& name) {
    // Try to load the cache the first time that we want to get a file
    if (this->_cache.size() == 0 && this->_last_modified == 0) {
        try {
            this->refresh();
        } catch (std::system_error e) {
            // Fail silently as the file could not exist
        }
    }

    auto it = this->_cache.find(name);
    if (it != this->_cache.end()) return it->second;

    if (this->_mode != LocalContext::FileMode::local) {
        // If we are not in local mode and didn't file the cache, we can try to refresh it
        size_t last_cache_size = this->_cache.size();
        this->refresh();
        if (this->_cache.size() > last_cache_size) {
            it = this->_cache.find(name);
            if (it != this->_cache.end()) return it->second;
        }
    }

    return nullptr;
}

void RegularFileIndex::add(const std::shared_ptr<RegularFile>& file) {
    if (this->_mode != LocalContext::FileMode::local) throw std::logic_error("Error while trying to add entry to " + this->_filename + " in non local mode");

    // Try to load the cache the first time that we want to add an entry
    if (this->_cache.size() == 0 && this->_last_modified == 0) {
        try {
            this->refresh();
        } catch (std::system_error e) {
            // Fail silently as the file could not exist
        }
    }

    this->_cache[file->name()] = file;
    this->_write(file);
}

void RegularFileIndex::refresh() {
    Logger::debug << "Refresh index file " << this->_filename << " on mode " << (this->_mode == LocalContext::FileMode::local ? "local" : "remote")
                  << std::endl;
    struct stat stat_buf;
    if (stat(_path_on_disk.c_str(), &stat_buf) != 0) throw std::system_error(errno, std::system_category(), "Could not stat file index " + _path_on_disk);

    if (stat_buf.st_mtime > this->_last_modified || (size_t)stat_buf.st_size > this->_last_size) {
        Logger::debug << "Index file " << this->_filename << " has been modified, gogo" << std::endl;
        std::lock_guard<std::mutex> lock(this->_file_lock);

        this->_last_modified = stat_buf.st_mtime;

        FILE* f = fopen(_path_on_disk.c_str(), "ro");
        if (f == NULL) throw std::system_error(errno, std::system_category(), "Could not open file index " + _path_on_disk);

        if (this->_last_size > 0) {
            Logger::debug << "Seek to " << this->_last_size << std::endl;
            fseek(f, this->_last_size, SEEK_SET);
        }

        std::vector<std::shared_ptr<File>> read_files;
        FileSerializer::read<CSVFileSerializer>(f, read_files, false);
        for (auto it = read_files.begin(); it != read_files.end(); ++it) {
            auto file = *it;
            if (file->type() == File::Type::REGULAR_FILE) { this->_cache[file->name()] = std::dynamic_pointer_cast<RegularFile>(file); }
        }

        this->_last_size = ftell(f); // Keeps last read position, not really line size as the file could not be complete
        fclose(f);
    }
}

void RegularFileIndex::_write(const std::shared_ptr<RegularFile>& file) {
    std::lock_guard<std::mutex> lock(this->_file_lock);
    if (this->_serializer == nullptr) {
        FILE* f = fopen(_path_on_disk.c_str(), "a");
        if (f == NULL) throw std::system_error(errno, std::system_category(), "Could not open file index " + _path_on_disk);
        this->_serializer = FileSerializer::writer<CSVFileSerializer>(f, true);
    }
    this->_serializer << file << FileSerializer::StreamOperation::flush;
}

} // namespace flocons