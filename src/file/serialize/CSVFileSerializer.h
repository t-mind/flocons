#ifndef _FLOCONS_FILE_CSV_FILE_SERIALIZER_H_
#define _FLOCONS_FILE_CSV_FILE_SERIALIZER_H_

#include "../../common.h"
#include "FileSerializer.h"

namespace flocons {
class CSVFileSerializer : public FileSerializer {
  public:
    CSVFileSerializer(Mode mode, char* dest, size_t max_size) : FileSerializer(mode, dest, max_size) {}
    CSVFileSerializer(Mode mode, char** dest, size_t* size) : FileSerializer(mode, dest, size) {}
    CSVFileSerializer(Mode mode, FILE* file, bool must_close_file) : FileSerializer(mode, file, must_close_file) {}

    virtual size_t read(std::shared_ptr<File>& file);
    virtual size_t read(std::vector<std::shared_ptr<File>>& files) { return FileSerializer::read(files); }
    virtual size_t write(const std::shared_ptr<File>& file);
    virtual size_t write(const std::vector<std::shared_ptr<File>>& files) { return FileSerializer::write(files); }
};
} // namespace flocons

#endif // !_FLOCONS_FILE_CSV_FILE_SERIALIZER_H_