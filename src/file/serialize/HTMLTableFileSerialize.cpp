#include "../RegularFile.h"
#include "HTMLTableFileSerializer.h"

namespace flocons {
HTMLTableFileSerializer::~HTMLTableFileSerializer() {
    if (!this->_error && this->_processed > 0) this->writeFormat("</table</body></document></html>");
}

size_t HTMLTableFileSerializer::read(std::shared_ptr<File>& file) { return this->_processed; }

size_t HTMLTableFileSerializer::write(const std::shared_ptr<File>& file) {
    if (this->_processed == 0) this->writeFormat("<html><document><body><table>");
    this->writeFormat("<tr><td><a href=\"%s\">%s</a></td>", file->name().c_str(), file->name().c_str());
    switch (file->type()) {
    case File::Type::REGULAR_FILE: {
        auto regular_file = std::dynamic_pointer_cast<RegularFile>(file);
        this->writeFormat("<td>%lu</td>", regular_file->size());
        break;
    }
    case File::Type::DIRECTORY: this->writeFormat("<td>-</td>"); break;
    }

    static __thread struct tm tm;
    static __thread char time_buffer[256];
    time_t time = file->modificationTime();
    gmtime_r(&time, &tm);
    strftime(time_buffer, 256, "%a, %d %b %Y %H:%M:%S %Z", &tm);
    this->writeFormat("<td>%s</td>", time_buffer);

    this->writeFormat("</tr>");
    return this->_processed;
}
} // namespace flocons