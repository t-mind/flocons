#include "RegularFileContainer.h"
#include "../Logger.h"
#include "../file/access/DataContainerRegularFileAccessor.h"
#include "../file/access/DataProxyRegularFileAccessor.h"
#include <archive.h>
#include <archive_entry.h>
#include <stdio.h>

namespace flocons {

RegularFileContainer::RegularFileContainer(const std::shared_ptr<LocalContext> context, const std::string& filename, int order, const Path& dir_path,
                                           LocalContext::FileMode mode)
    : _context(context), _mode(mode), _filename(filename), _order(order), _dir_path(dir_path), _index(context, "index" + filename.substr(5), mode, dir_path) {
    _path_on_disk = context->path() / dir_path / (filename + ".tar");
    Logger::debug << "Create regular file container " << filename << " with mode " << (mode == LocalContext::FileMode::local ? "local" : "remote") << std::endl;
}

RegularFileContainer::~RegularFileContainer() {
    if (this->_append_file_ptr != NULL) fclose(this->_append_file_ptr);
}

void RegularFileContainer::_addRegularFileAccessor(const std::shared_ptr<RegularFile>& file) {
    file->accessor(std::make_unique<DataProxyRegularFileAccessor>(std::bind(&RegularFileContainer::getRegularFileContent, this, file)));
}

std::shared_ptr<RegularFile> RegularFileContainer::writeRegularFile(const std::string& path, size_t size, const char* data, mode_t mode) {
    std::lock_guard<std::mutex> lock(this->_container_lock);
    Logger::debug << "Write new regular file " << path << " in " << this->_filename << " of size " << size << std::endl;
    if (this->_mode != LocalContext::FileMode::local) throw std::logic_error("Error trying to write file container " + this->_filename + " in non local mode");

    if (this->_append_file_ptr == NULL) {
        this->_append_file_ptr = fopen(_path_on_disk.c_str(), "a");
        if (this->_append_file_ptr == NULL) throw std::system_error(errno, std::system_category(), "Could not open file container " + _path_on_disk);
    }

    fseek(this->_append_file_ptr, 0, SEEK_END);
    size_t position = ftell(this->_append_file_ptr);

    struct archive* a = archive_write_new();
    archive_write_set_format_ustar(a);
    archive_write_add_filter_none(a);
    archive_write_open_FILE(a, this->_append_file_ptr);

    struct archive_entry* entry = archive_entry_new();
    archive_entry_set_pathname(entry, path.c_str());
    archive_entry_set_size(entry, size);
    archive_entry_set_filetype(entry, AE_IFREG);
    archive_entry_set_perm(entry, mode);
    archive_write_header(a, entry);

    archive_write_data(a, data, size);

    archive_entry_free(entry);
    archive_write_close(a);
    archive_write_free(a);

    auto file = std::make_shared<RegularFile>(this->_dir_path / path, size, mode, time(NULL), position);
    this->_addRegularFileAccessor(file);
    this->_index.add(file);
    return file;
}

std::shared_ptr<RegularFile> RegularFileContainer::getRegularFile(const std::string& path) {
    auto entry = this->_index.get(path);
    if (entry != nullptr) this->_addRegularFileAccessor(entry);

    return entry;
}

char* RegularFileContainer::getRegularFileContent(const std::shared_ptr<RegularFile>& file) {
    Logger::debug << "Read regular file in " << this->_filename << " at address " << file->address() << " of size " << file->size() << std::endl;
    FILE* f = fopen(_path_on_disk.c_str(), "ro");
    if (f == NULL) throw std::system_error(errno, std::system_category(), "Could not open file container " + _path_on_disk);

    fseek(f, file->address(), SEEK_SET);

    struct archive* a = archive_read_new();
    archive_read_support_format_tar(a);
    archive_read_open_FILE(a, f);

    struct archive_entry* entry;
    archive_read_next_header(a, &entry);

    char* data = (char*)calloc(file->size(), 1);
    if (data == NULL) {
        archive_read_close(a);
        archive_read_free(a);
        throw std::system_error(errno, std::system_category(), "Could not instantiate memory while reading file container");
    }

    archive_read_data(a, data, file->size());

    // archive_entry_free(entry); // don't free entry, it was create by archive_read_next_header and then freed by archive
    archive_read_close(a);
    archive_read_free(a);
    fclose(f);

    return data;
}

std::shared_ptr<RegularFile> RegularFileContainer::getRegularFileFromRawContainer(const std::string& path) {
    std::shared_ptr<RegularFile> file = nullptr;

    struct archive* a = archive_read_new();
    archive_read_support_format_tar(a);
    if (archive_read_open_filename(a, _path_on_disk.c_str(), 10240) != ARCHIVE_OK) {
        // The file does not exist
        return nullptr;
    }

    struct archive_entry* entry;
    while (archive_read_next_header(a, &entry) != ARCHIVE_EOF) {
        auto name = archive_entry_pathname(entry);
        if (path == name) {
            size_t size = archive_entry_size(entry);
            mode_t mode = archive_entry_perm(entry);
            time_t modification_time = archive_entry_mtime(entry);

            file = std::make_shared<RegularFile>(path, size, mode, modification_time, archive_read_header_position(a));
            this->_addRegularFileAccessor(file);
            break;
        }
    }
    archive_read_close(a);
    archive_read_free(a);
    return file;
}

std::vector<std::shared_ptr<RegularFile>> RegularFileContainer::listRegularFiles() {
    this->_index.refresh();
    auto entries = this->_index.cache();
    std::vector<std::shared_ptr<RegularFile>> files(entries.size());
    for (auto it = entries.begin(); it != entries.end(); ++it) {
        auto file = it->second;
        this->_addRegularFileAccessor(file);
        files.push_back(file);
    }
    return files;
}

} // namespace flocons