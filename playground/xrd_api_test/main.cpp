#include <XrdCl/XrdClFileSystem.hh>
#include <iostream>
#include <vector>

// A test project to use xrd client API to list a directory
int main()
{
    const std::string serverUrl = "root://localhost"; // Replace with your XRootD server URL
    const std::string directory = "/tmp";             // Replace with the desired directory

    XrdCl::FileSystem fs(serverUrl);
    XrdCl::DirectoryList* directoryContent(nullptr);
    XrdCl::XRootDStatus status = fs.DirList(directory, XrdCl::DirListFlags::Flags::Stat, directoryContent);
    XrdCl::DirectoryList::ConstIterator iter = directoryContent->Begin();
    while (iter != directoryContent->End())
    {
        std::cout << (*iter)->GetName() << std::endl;
        iter++;
    }
    return 0;
}
