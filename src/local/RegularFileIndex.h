#ifndef _FLOCONS_LOCAL_REGULAR_FILE_INDEX_H_
#define _FLOCONS_LOCAL_REGULAR_FILE_INDEX_H_

#include "../common.h"
#include "../file/Path.h"
#include "../file/serialize/FileSerializer.h"
#include "LocalContext.h"

namespace flocons {

class RegularFileIndex {
  private:
    std::weak_ptr<LocalContext> _context;
    std::string _filename;
    LocalContext::FileMode _mode;
    Path _dir_path;
    Path _path_on_disk;
    std::unordered_map<std::string, std::shared_ptr<RegularFile>> _cache;
    std::unique_ptr<FileSerializer::OStream> _serializer;

    time_t _last_modified = 0;
    size_t _last_size = 0;

    std::mutex _file_lock;
    void _write(const std::shared_ptr<RegularFile>& entry);

  public:
    RegularFileIndex(const std::shared_ptr<LocalContext>& context, const std::string& filename, LocalContext::FileMode mode, const Path& dir_path);
    int count();
    void refresh();
    const std::unordered_map<std::string, std::shared_ptr<RegularFile>>& cache() const { return this->_cache; }
    std::shared_ptr<RegularFile> get(const std::string& name);
    void add(const std::shared_ptr<RegularFile>& file);
};

} // namespace flocons

#endif // !_FLOCONS_LOCAL_REGULAR_FILE_INDEX_H_