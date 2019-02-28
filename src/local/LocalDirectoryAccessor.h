#ifndef _FLOCONS_LOCAL_DIRECTORY_H_
#define _FLOCONS_LOCAL_DIRECTORY_H

#include "../file/Directory.h"
#include "../file/RegularFile.h"
#include "../file/access/DirectoryAccessor.h"
#include "LocalContext.h"
#include "RegularFileContainer.h"

namespace flocons {

class LocalDirectoryAccessor : public DirectoryAccessor {
  private:
    Path _path;
    std::weak_ptr<LocalContext> _context;
    std::unordered_map<std::string, std::shared_ptr<RegularFileContainer>> _file_containers;
    std::shared_ptr<RegularFileContainer> _current_writable_file_container;
    std::vector<std::shared_ptr<RegularFileContainer>> _refresh_file_containers();
    void _new_writable_file_container();
    std::recursive_mutex _container_lock;
    std::mutex _directory_cache_lock;
    std::shared_ptr<Directory> _createDirectoryObject(const Path& full_path, mode_t mode, time_t modification_time);

  public:
    LocalDirectoryAccessor(const std::shared_ptr<LocalContext>& context, const Path& path) : _path(path), _context(context) {}
    virtual std::shared_ptr<File> getFile(const std::string& path);
    std::shared_ptr<RegularFile> getRegularFile(const std::string& path);
    virtual std::shared_ptr<Directory> createDirectory(const std::string& path, mode_t mode);
    virtual std::shared_ptr<RegularFile> createRegularFile(const std::string& path, const char* data, size_t size, mode_t mode);
    virtual std::vector<std::shared_ptr<File>> listFiles();
};

} // namespace flocons

#endif // !_FLOCONS_LOCAL_DIRECTORY_H_
