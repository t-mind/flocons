#include "CSVFileSerializer.h"
#include "../../Logger.h"
#include "../RegularFile.h"

namespace flocons {
static thread_local auto logger = Logger::category("serializer");

size_t CSVFileSerializer::read(std::shared_ptr<File>& file) {
    static __thread size_t position;
    static __thread size_t size;
    static __thread mode_t mode;
    static __thread time_t modification_time;
    static __thread char name[NAME_MAX];
    if (this->readFormattedLine("%lu;%lu;%o;%ld;%s", &position, &size, &mode, &modification_time, name) == 5) {
        logger->debug << "Found index entry " << name << std::endl;
        file = std::make_shared<RegularFile>(name, size, mode, modification_time, position);
    } else {
        file = nullptr;
    }
    return this->_processed;
}

size_t CSVFileSerializer::write(const std::shared_ptr<File>& file) {
    switch (file->type()) {
    case File::Type::REGULAR_FILE: {
        auto regular_file = std::dynamic_pointer_cast<RegularFile>(file);
        this->writeFormat("%lu;%lu;%o;%ld;%s\n", file->address(), regular_file->size(), file->mode(), file->modificationTime(), file->name().c_str());
        break;
    }
    case File::Type::DIRECTORY: this->writeFormat("<td>-</td>"); break;
    }
    return this->_processed;
}
} // namespace flocons