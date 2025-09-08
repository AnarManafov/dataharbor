"""
This script builds RPM packages for the dataharbor application.

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
    python build_rpm.py [-b] [-f] [-v VERSION]

    Options:
        -b, --backend   Build backend package.
        -f, --frontend  Build frontend package.
        -v, --version   Specify the version to use for the package.

If no options are specified, both backend and frontend packages will be built.
"""
import argparse
import subprocess
import os
import platform
import sys
import json
import glob
import shutil


def get_package_version(package_json_path):
    """Get the version from a package.json file."""
    try:
        with open(package_json_path, 'r') as f:
            package_data = json.load(f)
            return package_data.get('version', '0.1.0')
    except Exception as e:
        print(f"Failed to read package.json at {package_json_path}: {e}")
        return '0.1.0'


def build_package(app_name, source_dir, spec_file, version, nginx_conf_path=None):
    build_dir = os.path.expanduser("~/rpmbuild")
    release_notes_file = "RELEASE_NOTES.md"
    # Create temp directory for final RPMs
    tmp_rpm_dir = "/tmp/all-rpms"
    os.makedirs(tmp_rpm_dir, exist_ok=True)

    # Create RPM build directories if they don't exist
    os.makedirs(f"{build_dir}/BUILD", exist_ok=True)
    os.makedirs(f"{build_dir}/RPMS", exist_ok=True)
    os.makedirs(f"{build_dir}/SOURCES", exist_ok=True)
    os.makedirs(f"{build_dir}/SPECS", exist_ok=True)
    os.makedirs(f"{build_dir}/SRPMS", exist_ok=True)

    print(f"Building the {app_name} application...")

    # Get target architecture from environment or detect it
    target_arch = os.environ.get('GOARCH', platform.machine())
    if target_arch == 'x86_64':
        target_arch = 'amd64'
    elif target_arch in ['aarch64', 'arm64']:
        target_arch = 'arm64'

    print(f"Target architecture: {target_arch}")

    # Store current directory to return to it later
    original_dir = os.getcwd()

    # Change to the source directory
    os.chdir(source_dir)

    # Build the application based on its type
    if os.path.isfile("package.json"):
        subprocess.run(["npm", "install"], check=True)
        subprocess.run(["npm", "run", "build"], check=True)
    elif os.path.isfile("go.mod"):
        go_env = os.environ.copy()
        go_env["GOPATH"] = os.path.expanduser("~/go")

        # Check if there's a main.go file in the current directory
        if os.path.isfile("main.go"):
            # If main.go exists, build just the main package to a single binary with static linking
            print(
                f"Building main package to {app_name} with static linking for {target_arch}...")

            # Prepare ldflags for version injection
            ldflags = f"-s -w -X github.com/AnarManafov/dataharbor/app/config.Version={version}"

            # Try to get git commit hash
            try:
                git_commit = subprocess.check_output(
                    ["git", "rev-parse", "--short", "HEAD"],
                    stderr=subprocess.DEVNULL
                ).decode().strip()
                ldflags += f" -X github.com/AnarManafov/dataharbor/app/config.GitCommit={git_commit}"
            except:
                pass

            # Add build time
            import datetime
            build_time = datetime.datetime.utcnow().strftime("%Y-%m-%dT%H:%M:%SZ")
            ldflags += f" -X github.com/AnarManafov/dataharbor/app/config.BuildTime={build_time}"

            build_env = {
                **go_env,
                "CGO_ENABLED": "0",
                "GOOS": "linux",
                "GOARCH": target_arch
            }

            subprocess.run(["go", "build", "-v", f"-ldflags={ldflags}", "-o",
                           app_name, "."], check=True, env=build_env)
        else:
            # For multiple packages, use a more specific approach
            # First identify the main package
            main_pkg = None
            for root, dirs, files in os.walk("."):
                if "main.go" in files:
                    main_pkg = os.path.relpath(root, ".")
                    break

            if main_pkg:
                print(
                    f"Building main package from {main_pkg} to {app_name} with static linking for {target_arch}...")

                # Prepare ldflags for version injection
                ldflags = f"-s -w -X github.com/AnarManafov/dataharbor/app/config.Version={version}"

                # Try to get git commit hash
                try:
                    git_commit = subprocess.check_output(
                        ["git", "rev-parse", "--short", "HEAD"],
                        stderr=subprocess.DEVNULL
                    ).decode().strip()
                    ldflags += f" -X github.com/AnarManafov/dataharbor/app/config.GitCommit={git_commit}"
                except:
                    pass

                # Add build time
                import datetime
                build_time = datetime.datetime.utcnow().strftime("%Y-%m-%dT%H:%M:%SZ")
                ldflags += f" -X github.com/AnarManafov/dataharbor/app/config.BuildTime={build_time}"

                build_env = {
                    **go_env,
                    "CGO_ENABLED": "0",
                    "GOOS": "linux",
                    "GOARCH": target_arch
                }

                subprocess.run(["go", "build", "-v", f"-ldflags={ldflags}", "-o",
                               app_name, main_pkg], check=True, env=build_env)
            else:
                print("Error: No main.go file found in any package")
                return False
    else:
        print("Unknown project type. Exiting.")
        return

    # Return to the original directory
    os.chdir(original_dir)

    print(f"Preparing source files for {app_name}...")

    # Handle frontend (with dist directory)
    if "frontend" in app_name and os.path.isdir(f"{source_dir}/dist"):
        # Create version directory in SOURCES
        dest_dir = f"{build_dir}/SOURCES/{app_name}-{version}"
        os.makedirs(dest_dir, exist_ok=True)

        # Copy the dist contents
        subprocess.run(
            ["cp", "-r", f"{source_dir}/dist/.", dest_dir], check=True)

        # Copy nginx.conf to SOURCES directory if provided
        if nginx_conf_path and os.path.exists(nginx_conf_path):
            print("Copying nginx.conf to SOURCES directory...")
            subprocess.run(
                ["cp", nginx_conf_path, f"{build_dir}/SOURCES/"], check=True)

        # Create source tarball
        orig_dir = os.getcwd()
        os.chdir(f"{build_dir}/SOURCES")
        subprocess.run(
            ["tar", "czvf", f"{app_name}-{version}.tar.gz", f"{app_name}-{version}"], check=True)
        os.chdir(orig_dir)

    # Handle backend (with binary)
    elif "backend" in app_name:
        # For backend, we need to create our own source directory structure
        dest_dir = f"{build_dir}/SOURCES/{app_name}-{version}"
        os.makedirs(dest_dir, exist_ok=True)

        # Copy the binary to SOURCES
        if os.path.exists(f"{source_dir}/{app_name}"):
            shutil.copy(f"{source_dir}/{app_name}", f"{build_dir}/SOURCES/")
            # Create source tarball for backend
            orig_dir = os.getcwd()
            os.chdir(f"{build_dir}/SOURCES")
            subprocess.run(
                ["tar", "czvf", f"{app_name}-{version}.tar.gz", f"{app_name}"], check=True)
            os.chdir(orig_dir)
        else:
            print(f"Error: Binary {app_name} not found in {source_dir}")
            return False

    # Copy the spec file to the SPECS directory
    subprocess.run(["cp", spec_file, f"{build_dir}/SPECS/"], check=True)

    print("Generating changelog...")
    # Generate changelog
    script_dir = os.path.dirname(os.path.realpath(__file__))
    changelog_script = os.path.join(script_dir, "generate_changelog.py")
    release_notes_path = os.path.join(source_dir, release_notes_file)
    spec_path = os.path.join(build_dir, "SPECS", os.path.basename(spec_file))
    subprocess.run(["python3", changelog_script, spec_path,
                   release_notes_path], check=True)

    print(
        f"Building the RPM package for version {version} and architecture {target_arch}...")
    try:
        # Map Go architectures to RPM architectures
        rpm_arch = target_arch
        if target_arch == 'amd64':
            rpm_arch = 'x86_64'
        elif target_arch == 'arm64':
            rpm_arch = 'aarch64'

        subprocess.run(["rpmbuild", "-ba", f"{build_dir}/SPECS/{os.path.basename(spec_file)}",
                        f"--define", f"_version {version}",
                        f"--target", rpm_arch], check=True)

        # Copy built RPMs to /tmp/all-rpms directory
        for rpm_path in glob.glob(f"{build_dir}/RPMS/*/{app_name}-*.rpm"):
            print(f"Copying {rpm_path} to {tmp_rpm_dir}")
            shutil.copy(rpm_path, tmp_rpm_dir)
        for rpm_path in glob.glob(f"{build_dir}/RPMS/{app_name}-*.rpm"):
            print(f"Copying {rpm_path} to {tmp_rpm_dir}")
            shutil.copy(rpm_path, tmp_rpm_dir)

        print(
            f"Successfully built RPM package for {app_name} version {version}")
    except subprocess.CalledProcessError as e:
        print(f"Failed to create RPM package. Error: {e}")
        return False

    return True


def main():
    parser = argparse.ArgumentParser(description="Build RPM packages.")
    parser.add_argument("-b", "--backend", action="store_true",
                        help="Build backend package")
    parser.add_argument("-f", "--frontend", action="store_true",
                        help="Build frontend package")
    parser.add_argument("-v", "--version",
                        help="Version to use for the package")
    args = parser.parse_args()

    # Get version from command line or package.json
    if args.version:
        version = args.version
    else:
        # Try to get from root package.json
        version = get_package_version("package.json")

    app_name_backend = "dataharbor-backend"
    source_dir_backend = "app"
    spec_file_backend = f"packaging/{app_name_backend}.spec"

    app_name_frontend = "dataharbor-frontend"
    source_dir_frontend = "web"
    spec_file_frontend = f"packaging/{app_name_frontend}.spec"
    nginx_conf_path = "web/nginx.conf"

    if not args.backend and not args.frontend:
        args.backend = True
        args.frontend = True

    success = True

    if args.backend:
        print(f"Building backend RPM with version {version}")
        if not build_package(app_name_backend, source_dir_backend,
                             spec_file_backend, version):
            success = False

    if args.frontend:
        print(f"Building frontend RPM with version {version}")
        if not build_package(app_name_frontend, source_dir_frontend,
                             spec_file_frontend, version, nginx_conf_path):
            success = False

    # If no RPMs were built successfully, exit with an error code
    if not success:
        sys.exit(1)


if __name__ == "__main__":
    main()
