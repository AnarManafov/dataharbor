"""
This script builds RPM packages for the data-lake-ui application.

Functions:
    build_package(app_name, source_dir, spec_file, version):
        Builds the RPM package for the specified application.
        Parameters:
            app_name (str): The name of the application.
            source_dir (str): The source directory of the application.
            spec_file (str): The path to the RPM spec file.
            version (str): The version of the application.

    main():
        Parses command-line arguments and builds the specified packages.

Usage:
    python build_rpm.py [-b] [-f]

    Options:
        -b, --backend   Build backend package.
        -f, --frontend  Build frontend package.

If no options are specified, both backend and frontend packages will be built.
"""
import argparse
import subprocess
import os


def build_package(app_name, source_dir, spec_file, version, nginx_conf_path=None):
    build_dir = os.path.expanduser("~/rpmbuild")
    release_notes_file = "RELEASE_NOTES.md"

    os.makedirs(f"{build_dir}/BUILD", exist_ok=True)
    os.makedirs(f"{build_dir}/RPMS", exist_ok=True)
    os.makedirs(f"{build_dir}/SOURCES", exist_ok=True)
    os.makedirs(f"{build_dir}/SPECS", exist_ok=True)
    os.makedirs(f"{build_dir}/SRPMS", exist_ok=True)

    print("Building the application...")
    os.chdir(source_dir)
    if os.path.isfile("package.json"):
        subprocess.run(["npm", "install"], check=True)
        subprocess.run(["npm", "run", "build"], check=True)
    elif os.path.isfile("go.mod"):
        go_env = os.environ.copy()
        go_env["GOOS"] = "linux"
        go_env["GOARCH"] = "amd64"
        go_env["GOPATH"] = os.path.expanduser("~/go")
        subprocess.run(["go", "build", "-o", app_name], check=True, env=go_env)
    else:
        print("Unknown project type. Exiting.")
        return

    os.chdir("..")

    print("Copying binaries to SOURCES directory...")
    if os.path.isdir(f"{source_dir}/dist"):
        subprocess.run(["cp", "-r", f"{source_dir}/dist",
                       f"{build_dir}/SOURCES/{app_name}-{version}"], check=True)
    else:
        subprocess.run(["cp", f"{source_dir}/{app_name}",
                       f"{build_dir}/SOURCES/"], check=True)

    print("Copying nginx.conf to SOURCES directory...")
    if nginx_conf_path:
        subprocess.run(["cp", nginx_conf_path, f"{
                       build_dir}/SOURCES/"], check=True)

    print("Creating source tarballs...")
    if os.path.isdir(f"{source_dir}/dist"):
        subprocess.run(["tar", "czvf", f"{build_dir}/SOURCES/{app_name}-{version}.tar.gz",
                       "-C", f"{build_dir}/SOURCES", f"{app_name}-{version}"], check=True)
    else:
        subprocess.run(
            ["tar", "czvf", f"{build_dir}/SOURCES/{app_name}-{version}.tar.gz", "-C", source_dir, app_name], check=True)

    print("Copying spec file...")
    subprocess.run(["cp", spec_file, f"{build_dir}/SPECS/"], check=True)

    print("Generating changelog...")
    changelog_script_path = os.path.join(
        os.path.dirname(__file__), "generate_changelog.py")
    subprocess.run(["python3", changelog_script_path,
                   f"{build_dir}/SPECS/{os.path.basename(spec_file)}", f"{source_dir}/{release_notes_file}"], check=True)

    print("Building the RPM package...")
    result = subprocess.run(
        ["rpmbuild", "-ba", f"{build_dir}/SPECS/{os.path.basename(spec_file)}"])
    if result.returncode == 0:
        print("RPM package created successfully.")
    else:
        print("Failed to create RPM package.")
        return


def main():
    parser = argparse.ArgumentParser(description="Build RPM packages.")
    parser.add_argument("-b", "--backend", action="store_true",
                        help="Build backend package")
    parser.add_argument("-f", "--frontend", action="store_true",
                        help="Build frontend package")
    args = parser.parse_args()

    version_backend = "0.6.0"
    app_name_backend = "data-lake-ui-backend"
    source_dir_backend = "app"
    spec_file_backend = f"packaging/{app_name_backend}.spec"

    version_frontend = "0.6.0"
    app_name_frontend = "data-lake-ui-frontend"
    source_dir_frontend = "web"
    spec_file_frontend = f"packaging/{app_name_frontend}.spec"
    nginx_conf_path = "web/nginx.conf"

    if not args.backend and not args.frontend:
        args.backend = True
        args.frontend = True

    if args.backend:
        build_package(app_name_backend, source_dir_backend,
                      spec_file_backend, version_backend)

    if args.frontend:
        build_package(app_name_frontend, source_dir_frontend,
                      spec_file_frontend, version_frontend, nginx_conf_path)


if __name__ == "__main__":
    main()
