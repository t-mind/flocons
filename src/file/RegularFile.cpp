#include "RegularFile.h"

namespace flocons {

static const std::map<std::string, const std::string> _mime_types{
    {"jpg", JPEG_MIME_TYPE},
    {"jpeg", JPEG_MIME_TYPE},
    {"mp4", MP4_MIME_TYPE},
    {"txt", TXT_MIME_TYPE},
};
static const std::string _default_file_mime_type(DEFAULT_FILE_MIME_TYPE);

char* RegularFile::data() {
    if (this->_accessor != nullptr) return this->_accessor->data();
    return NULL;
}

const std::string& RegularFile::getMimeType(const std::string& extension) {
    std::string extension_to_lower = extension;
    std::for_each(extension_to_lower.begin(), extension_to_lower.end(), [](char& c) { c = ::tolower(c); });

    auto it = _mime_types.find(extension_to_lower);
    return it != _mime_types.end() ? it->second : _default_file_mime_type;
}

} // namespace flocons