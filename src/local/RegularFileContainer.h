#ifndef _FLOCONS_LOCAL_REGULAR_FILE_CONTAINER_H
#define _FLOCONS_LOCAL_REGULAR_FILE_CONTAINER_H

#include "../common.h"
#include "LocalContext.h"
#include "RegularFileIndex.h"

namespace flocons {

class RegularFileContainer {
  private:
    std::weak_ptr<LocalContext> _context;
    LocalContext::FileMode _mode;
    std::string _filename;
    int _order;
    Path _dir_path;
    Path _path_on_disk;
    RegularFileIndex _index;

    FILE* _append_file_ptr = NULL;

    std::mutex _container_lock;

    void _addRegularFileAccessor(const std::shared_ptr<RegularFile>& file);

  public:
    RegularFileContainer(const std::shared_ptr<LocalContext> context, const std::string& filename, int order, const Path& dir_path,
                         LocalContext::FileMode mode);
    virtual ~RegularFileContainer();
    LocalContext::FileMode mode() const { return this->_mode; }
    int order() const { return this->_order; }
    int count() { return this->_index.count(); }
    std::shared_ptr<RegularFile> writeRegularFile(const std::string& path, size_t size, const char* data, mode_t mode);
    std::shared_ptr<RegularFile> getRegularFile(const std::string& path);
    char* getRegularFileContent(const std::shared_ptr<RegularFile>& file);
    std::shared_ptr<RegularFile> getRegularFileFromRawContainer(const std::string& path);
    std::vector<std::shared_ptr<RegularFile>> listRegularFiles();
};

} // namespace flocons

#endif // !_FLOCONS_LOCAL_REGULAR_FILE_CONTAINER_H