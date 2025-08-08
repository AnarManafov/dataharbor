#!/usr/bin/env python3
"""
Unified Changelog Management Script

A single tool to handle all changelog operations:
1. Generate a changelog for a specific version
2. Generate a full changelog from all tags
3. Update an existing changelog with a new version
4. Consolidate duplicate entries in a changelog

Usage:
  python changelog.py generate --from-tag TAG --to-tag TAG [--version VERSION] [--output FILE] [--dry-run] [--no-consolidate]
  python changelog.py update --version VERSION [--from-tag TAG] [--to-tag TAG] [--output FILE] [--dry-run] [--no-consolidate]
  python changelog.py full [--output FILE] [--no-consolidate]
  python changelog.py extract --version VERSION --output-file FILE

Options:
  --version       Version to add (e.g. "1.2.3")
  --from-tag      Starting tag for commits (default: latest tag)
  --to-tag        Ending reference (tag or commit) (default: HEAD)
  --output        Output file path (default: CHANGELOG.md in repo root)
  --dry-run       Don't write to file, just print what would be added
  --no-consolidate Disable automatic consolidation of duplicate entries
  --extract       Extract the content for a specific version
  --output-file   File to write extracted content to
"""

import argparse
import os
import re
import subprocess
from collections import defaultdict
from datetime import date, datetime
from typing import Dict, List, Optional, Tuple, Any


# ---- Utility Functions ----

def run_git_command(command: List[str]) -> str:
    """Run a git command and return its output."""
    try:
        return subprocess.check_output(["git"] + command, text=True).strip()
    except subprocess.CalledProcessError as e:
        print(f"Error running git command: {e}")
        return ""


def get_repo_root() -> str:
    """Get the root directory of the Git repository."""
    return run_git_command(["rev-parse", "--show-toplevel"])


def get_latest_tag() -> str:
    """Get the latest tag from the Git repository."""
    try:
        return run_git_command(["describe", "--tags", "--abbrev=0"])
    except subprocess.CalledProcessError:
        # No tags exist, use initial commit
        print("No previous tags found, using first commit")
        return run_git_command(["rev-list", "--max-parents=0", "HEAD"])


def get_tag_date(tag: str) -> str:
    """Get the date of a tag in YYYY-MM-DD format."""
    try:
        # Get the tag's commit date
        date_timestamp = run_git_command(["log", "-1", "--format=%ct", tag])
        if date_timestamp:
            # Convert to YYYY-MM-DD format
            date_obj = datetime.fromtimestamp(int(date_timestamp))
            return date_obj.strftime("%Y-%m-%d")
    except Exception as e:
        print(f"Error getting date for tag {tag}: {e}")

    # Return today's date if we can't get the tag date
    return date.today().strftime("%Y-%m-%d")


def get_commit_url(hash_val: str) -> str:
    """Get the URL to a commit in the repository."""
    try:
        remote_url = run_git_command(["config", "--get", "remote.origin.url"])
        github_match = re.search(
            r'github\.com[:/]([^/]+)/([^/.]+)', remote_url)

        if github_match:
            owner, repo = github_match.groups()
            return f"https://github.com/{owner}/{repo}/commit/{hash_val}"
    except Exception:
        pass

    # Default fallback
    return f"#{hash_val}"


# ---- Commit Processing Functions ----

def get_commits(from_ref: str, to_ref: str = "HEAD") -> List[Dict]:
    """
    Get commits between two Git references and parse them into structured data.
    """
    # Use a custom format to get all the information we need in one go
    format_str = "%h%n%an%n%at%n%s%n%b%n--COMMIT--"
    log_cmd = ["log", f"{from_ref}..{to_ref}",
               f"--pretty=format:{format_str}", "--no-merges", "--reverse"]

    try:
        log_output = run_git_command(log_cmd)
        if not log_output:
            return []

        # Split by commit delimiter and process each commit
        raw_commits = log_output.split("--COMMIT--")
        commits = []

        for raw_commit in raw_commits:
            if not raw_commit.strip():
                continue

            # Split the raw commit into its components
            parts = raw_commit.strip().split("\n", 4)
            if len(parts) < 4:
                continue  # Skip malformed commits

            hash_val, author, timestamp, subject = parts[0:4]
            body = parts[4] if len(parts) > 4 else ""

            # Extract ticket reference
            ticket_ref = extract_ticket_reference(
                subject) or extract_ticket_reference(body)

            # If the ticket reference was found at the beginning of the subject,
            # remove it from the subject to avoid duplication
            if ticket_ref and subject.upper().startswith(f"{ticket_ref.upper()}:"):
                # Remove the ticket reference from the beginning of the subject
                subject = re.sub(f"^{ticket_ref}:\\s*", "",
                                 subject, flags=re.IGNORECASE)

            # Try to parse conventional commit format
            conv_match = re.match(
                r"^(feat|fix|perf|refactor|docs|chore|style|test|build|ci|revert)(?:\(([^)]+)\))?: (.+)$", subject)

            if conv_match:
                type_val, scope, message = conv_match.groups()
            else:
                type_val = "other"
                scope = None
                message = subject

            # Create a structured commit object
            commit = {
                "hash": hash_val,
                "author": author,
                "date": datetime.fromtimestamp(int(timestamp)).strftime("%Y-%m-%d"),
                "subject": subject,
                "body": body,
                "type": type_val,
                "scope": scope,
                "message": message,
                "ticket_ref": ticket_ref
            }

            commits.append(commit)

        return commits

    except subprocess.CalledProcessError as e:
        print(f"Error getting commits between {from_ref} and {to_ref}: {e}")
        return []


def extract_ticket_reference(text: str) -> Optional[str]:
    """
    Extract ticket references from text in formats:
    - Refs: <TicketID>
    - Ref: <TicketID>
    - <TicketID>: commit subject
    - Any line ending with (<TicketID>)

    Returns the ticket ID if found, None otherwise.
    """
    if not text:
        return None

    # Look for "Refs: <TicketID>" or "Ref: <TicketID>" pattern
    refs_match = re.search(
        r'(?:Refs|Ref):\s+([A-Z]+-\d+|GH-\d+)', text, re.IGNORECASE)
    if refs_match:
        return refs_match.group(1).upper()  # Normalize to uppercase

    # Look for "<TicketID>: commit subject" pattern at the beginning of the message
    prefix_match = re.match(r'^([A-Z]+-\d+|GH-\d+):\s+', text, re.IGNORECASE)
    if prefix_match:
        return prefix_match.group(1).upper()  # Normalize to uppercase

    # Look for lines ending with ticket references like "(...) (JIRA-123)"
    end_match = re.search(r'\(([A-Z]+-\d+|GH-\d+)\)\s*$', text, re.IGNORECASE)
    if end_match:
        return end_match.group(1).upper()  # Normalize to uppercase

    return None


def group_commits_by_type(commits: List[Dict]) -> Dict[str, Dict[str, List[str]]]:
    """Group commits by their conventional commit type."""
    types = {
        "feat": {"title": "### Added", "items": []},
        "fix": {"title": "### Fixed", "items": []},
        "perf": {"title": "### Performance Improvements", "items": []},
        "refactor": {"title": "### Changed", "items": []},
        "docs": {"title": "### Documentation", "items": []},
        "chore": {"title": "### Maintenance", "items": []},
        "style": {"title": "### Style", "items": []},
        "test": {"title": "### Tests", "items": []},
        "build": {"title": "### Build", "items": []},
        "ci": {"title": "### CI", "items": []},
        "revert": {"title": "### Reverted", "items": []},
        "other": {"title": "### Other", "items": []}
    }

    for commit in commits:
        commit_type = commit["type"]
        message = commit["message"]
        scope = commit["scope"]
        hash_val = commit["hash"]
        ticket_ref = commit["ticket_ref"]

        # Format the message
        if scope:
            base_message = f"{message} ({scope})"
        else:
            base_message = message

        # Add ticket reference if available and not already in the message
        if ticket_ref and ticket_ref.upper() not in base_message.upper():
            formatted_message = f"- {base_message} [{ticket_ref}] ({hash_val})"
        else:
            formatted_message = f"- {base_message} ({hash_val})"

        # Add to the appropriate type
        if commit_type in types:
            types[commit_type]["items"].append(formatted_message)
        else:
            types["other"]["items"].append(formatted_message)

    return types


# ---- Changelog Generation Functions ----

def generate_version_section(version: str, tag_date: str, commits_by_type: Dict[str, Dict[str, List[str]]]) -> str:
    """Generate a changelog section for a specific version."""
    version_content = f"## [{version}] - {tag_date}\n\n"

    # Add each type section that has commits
    for type_data in commits_by_type.values():
        if type_data["items"]:
            # Add section title with blank lines before and after the list
            version_content += f"{type_data['title']}\n\n" + \
                "\n".join(type_data["items"]) + "\n\n"

    # If no commits, add a note
    if all(not type_data["items"] for type_data in commits_by_type.values()):
        version_content += "No significant changes.\n\n"

    return version_content


def update_changelog_file(changelog_path: str, version: str, version_content: str, no_consolidate: bool = False) -> bool:
    """Update the changelog file with a new version section."""
    try:
        # Define the standard header
        standard_header = """# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

"""

        # Check if file exists
        if os.path.exists(changelog_path):
            with open(changelog_path, 'r', encoding='utf-8') as file:
                content = file.read()

                # Check if version already exists in changelog
                if f"## [{version}]" in content:
                    print(
                        f"Version {version} already exists in the changelog. Skipping update.")
                    return False

                # Find the first version section to preserve existing versions
                version_pattern = r'^## \['
                version_match = re.search(
                    version_pattern, content, re.MULTILINE)

                if version_match:
                    # Extract existing versions (everything from first version onward)
                    existing_versions = content[version_match.start():]
                    # Build new content: standard header + new version + existing versions
                    new_content = standard_header + version_content + '\n' + existing_versions
                else:
                    # No existing versions, just header + new version
                    new_content = standard_header + version_content
        else:
            # Create new changelog with standard header
            new_content = standard_header + version_content

        # Perform consolidation if not disabled
        if not no_consolidate:
            print("Automatically consolidating duplicate entries...")
            sections = parse_changelog(new_content)
            if sections:
                new_content = write_consolidated_changelog(
                    new_content, sections)

        # Write back to file
        with open(changelog_path, 'w', encoding='utf-8') as file:
            file.write(new_content)
        return True
    except Exception as e:
        print(f"Error updating changelog: {e}")
        return False


# ---- Changelog Consolidation Functions ----

def parse_changelog(content: str) -> List[Tuple[str, str, List[str]]]:
    """
    Parse the changelog content into sections.

    Returns a list of tuples: (version_header, date, [list of entries])
    """
    # Regex to find version headers like "## [0.13.7] - 2025-04-25"
    version_pattern = re.compile(r'^## \[(.*?)\](?: - (.*?))?$', re.MULTILINE)

    # Find all version headers
    version_matches = list(version_pattern.finditer(content))

    sections = []

    # Process each version section
    for i, match in enumerate(version_matches):
        version = match.group(1)
        date = match.group(2) if match.group(2) else ""
        version_header = match.group(0)

        # Determine the end of this section
        start_pos = match.end()
        if i < len(version_matches) - 1:
            end_pos = version_matches[i + 1].start()
        else:
            end_pos = len(content)

        # Extract the section content
        section_content = content[start_pos:end_pos]

        # Parse entries within this section
        entries = []
        current_subsection = None

        for line in section_content.strip().split('\n'):
            line = line.strip()

            # Skip empty lines
            if not line:
                continue

            # Check for subsection headers like "### Added" or "### Fixed"
            if line.startswith('### '):
                current_subsection = line
                entries.append(line)
            # Regular entries start with "- "
            elif line.startswith('- '):
                entries.append(line)

        sections.append((version_header, date, entries))

    return sections


def consolidate_entries(entries: List[str]) -> List[str]:
    """
    Consolidate duplicate entries in a section.
    Duplicates are identified by identical text before the commit hash.
    """
    if not entries:
        return []

    # Group entries by subsection
    subsections = []
    current_subsection = None
    current_entries = []

    for entry in entries:
        if entry.startswith('### '):
            # If we have a previous subsection, add it to the results
            if current_subsection is not None:
                subsections.append((current_subsection, current_entries))

            # Start a new subsection
            current_subsection = entry
            current_entries = []
        else:
            current_entries.append(entry)

    # Add the last subsection
    if current_subsection is not None:
        subsections.append((current_subsection, current_entries))

    # Process each subsection
    result = []
    for subsection, subentries in subsections:
        result.append(subsection)

        # Group entries by text (excluding the hash part)
        entry_groups = defaultdict(list)

        for entry in subentries:
            # Pattern to match entries with commit hashes like "- Text (hash)"
            match = re.match(r'- (.*?) \(([a-f0-9]+)\)$', entry)

            if match:
                text = match.group(1)
                commit_hash = match.group(2)
                entry_groups[text].append(commit_hash)
            else:
                # If the entry doesn't match the pattern, keep it as is
                entry_groups[entry].append(None)

        # Generate consolidated entries
        for text, hashes in entry_groups.items():
            if None in hashes:
                # If any entry didn't have a hash, keep it as is
                result.append(text)
            elif len(hashes) == 1:
                # Single entry, no consolidation needed
                result.append(f"- {text} ({hashes[0]})")
            else:
                # Multiple entries, consolidate the hashes
                if len(hashes) > 5:
                    # If there are many hashes, just show the count
                    result.append(f"- {text} ({len(hashes)} commits)")
                else:
                    # Show all hashes for smaller groups
                    hash_list = ", ".join(hashes)
                    result.append(f"- {text} ({hash_list})")

    return result


def write_consolidated_changelog(content: str, sections: List[Tuple[str, str, List[str]]]) -> str:
    """
    Write the consolidated changelog back to a string.
    """
    # Find the part of the content before the first version header
    first_version_header = sections[0][0] if sections else None
    header_content = content.split(first_version_header)[
        0] if first_version_header else content

    # Build the new content
    new_content = header_content.rstrip() + "\n\n"

    for i, (version_header, date, entries) in enumerate(sections):
        # Add the version header
        new_content += f"{version_header}\n\n"

        # Process entries to ensure proper spacing between section headers and lists
        consolidated = consolidate_entries(entries)
        current_section = None

        for entry in consolidated:
            if entry.startswith("### "):
                # Add extra newline after previous section if there was one
                if current_section:
                    new_content += "\n"

                # Add section header and a newline after it
                new_content += f"{entry}\n\n"
                current_section = entry
            else:
                # Add list item
                new_content += f"{entry}\n"

        # Add spacing between versions
        if i < len(sections) - 1:
            new_content += "\n"
        else:
            new_content += "\n"

    return new_content


# ---- Full Changelog Generation Functions ----

def get_all_version_tags() -> List[Tuple[str, str]]:
    """
    Get all version tags in the format vX.Y.Z sorted by version.
    Returns a list of tuples (tag, version_without_v)
    """
    # Get all tags
    all_tags = run_git_command(["tag", "--list"]).split("\n")

    # Filter for version tags with format vX.Y.Z
    version_tags = []
    for tag in all_tags:
        # Match standard version tags with format vX.Y.Z
        if re.match(r"^v\d+\.\d+\.\d+$", tag):
            # Extract version without 'v' prefix
            version = tag[1:]
            version_tags.append((tag, version))

    # Sort tags by version components (semver)
    def version_key(item):
        v = item[1].split('.')
        return [int(x) for x in v]

    version_tags.sort(key=version_key)
    return version_tags


def get_first_commit() -> str:
    """Get the hash of the first commit in the repository."""
    return run_git_command(["rev-list", "--max-parents=0", "HEAD"])


def check_for_untagged_commits(latest_tag: str) -> bool:
    """Check if there are any commits between the latest tag and HEAD."""
    commits = run_git_command(["log", f"{latest_tag}..HEAD", "--oneline"])
    return bool(commits.strip())


def extract_version_content(changelog_path: str, version: str, output_file: str) -> bool:
    """Extract the content for a specific version from CHANGELOG.md and write it to a file."""
    try:
        # Check if changelog exists
        if not os.path.exists(changelog_path):
            print(f"Error: {changelog_path} does not exist")
            return False

        # Read the changelog
        with open(changelog_path, "r", encoding="utf-8") as f:
            changelog = f.read()

        # Find the section for this version
        version_header = f"## [{version}]"
        version_start = changelog.find(version_header)

        if version_start == -1:
            print(
                f"Error: Version header '{version_header}' not found in {changelog_path}")
            return False

        # Find the next version section or end of file
        content_start_pos = version_start + len(version_header)
        content_end_pos = len(changelog)

        # Search for next version header after this version
        rest_of_changelog = changelog[content_start_pos:]
        next_version_match = re.search(
            r"^## \[", rest_of_changelog, re.MULTILINE)
        if next_version_match:
            content_end_pos = content_start_pos + next_version_match.start()

        # Extract the content
        version_content = changelog[content_start_pos:content_end_pos].strip()

        # Create directory if it doesn't exist
        output_dir = os.path.dirname(output_file)
        if output_dir and not os.path.exists(output_dir):
            os.makedirs(output_dir, exist_ok=True)

        # Write to file
        with open(output_file, "w", encoding="utf-8") as f:
            f.write(version_content)

        print(f"Extracted content for version {version} to {output_file}")
        return True

    except Exception as e:
        print(f"Error extracting version content: {e}")
        return False


# ---- Command Functions ----

def cmd_generate(args: Any) -> int:
    """Generate a changelog for a specific range of commits."""
    # Determine from_tag if not provided
    from_tag = args.from_tag
    if not from_tag:
        from_tag = get_latest_tag()
        if not from_tag:
            print("Could not determine the previous tag. Using initial commit.")
            from_tag = get_first_commit()

    to_tag = args.to_tag or "HEAD"

    # Get the version - either from args or from to_tag
    version = args.version
    if not version and to_tag != "HEAD":
        version = to_tag
        if version.startswith('v'):
            version = version[1:]

    if not version:
        print("Error: Version must be specified when generating a changelog to HEAD")
        return 1

    # Get release date - use today or tag date
    if to_tag == "HEAD":
        release_date = date.today().strftime("%Y-%m-%d")
    else:
        release_date = get_tag_date(to_tag)

    # Get commits and group by type
    commits = get_commits(from_tag, to_tag)
    commits_by_type = group_commits_by_type(commits)

    # Generate the changelog section
    section = generate_version_section(version, release_date, commits_by_type)

    if args.dry_run:
        print("\nChangelog section (dry run):\n")
        print(section)
    else:
        # Determine output path
        output_path = args.output
        if not output_path:
            repo_root = get_repo_root()
            output_path = os.path.join(repo_root, "CHANGELOG.md")

        # Update the file
        if update_changelog_file(output_path, version, section, args.no_consolidate):
            print(f"Successfully updated {output_path} with version {version}")
        else:
            print(f"Failed to update {output_path}")
            return 1

    return 0


def cmd_update(args: Any) -> int:
    """Update an existing changelog with a new version."""
    # This is similar to generate but with a focus on updating an existing file
    return cmd_generate(args)


def cmd_full(args: Any) -> int:
    """Generate a full changelog from all version tags."""
    # Get all version tags sorted by version
    version_tags = get_all_version_tags()

    if not version_tags:
        print("No version tags found in the repository.")
        return 1

    print(f"Found {len(version_tags)} version tags.")

    # Start with the header
    changelog_content = """# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

"""

    # Check if there are any untagged commits (between latest tag and HEAD)
    latest_tag = version_tags[-1][0]  # Get the latest tag
    has_untagged_commits = check_for_untagged_commits(latest_tag)

    # Only add the Unreleased section if there are untagged commits
    if has_untagged_commits:
        print("Found untagged commits. Adding Unreleased section.")
        # Get commits between latest tag and HEAD
        unreleased_commits = get_commits(latest_tag, "HEAD")
        if unreleased_commits:
            unreleased_by_type = group_commits_by_type(unreleased_commits)
            changelog_content += "## [Unreleased]\n\n"
            # Add each type section that has commits
            for type_data in unreleased_by_type.values():
                if type_data["items"]:
                    changelog_content += f"{type_data['title']}\n\n" + \
                        "\n".join(type_data["items"]) + "\n\n"

    # Process versions from newest to oldest for the changelog
    # Add sections in reverse chronological order (newest first)
    for i in range(len(version_tags) - 1, -1, -1):
        curr_tag, curr_version = version_tags[i]
        curr_date = get_tag_date(curr_tag)

        print(f"Processing version: {curr_tag}")

        # For the first version, use the first commit as the starting point
        if i == 0:
            prev_ref = get_first_commit()
            print(
                f"  First version - using first commit {prev_ref} as starting point")
        else:
            # For other versions, use the previous tag
            prev_tag, _ = version_tags[i-1]
            prev_ref = prev_tag
            print(f"  Using previous tag {prev_ref} as starting point")

        # Get commits between previous reference and current tag
        commits = get_commits(prev_ref, curr_tag)
        print(
            f"  Found {len(commits)} commits between {prev_ref} and {curr_tag}")

        # Group commits by type
        commits_by_type = group_commits_by_type(commits)

        # Generate section for this version
        version_section = generate_version_section(
            curr_version, curr_date, commits_by_type)
        changelog_content += version_section

    # Determine output path
    output_path = args.output
    if not output_path:
        repo_root = get_repo_root()
        output_path = os.path.join(repo_root, "CHANGELOG.md")

    # Consolidate duplicate entries if not disabled
    if not args.no_consolidate:
        print("Automatically consolidating duplicate entries...")
        sections = parse_changelog(changelog_content)
        if sections:
            changelog_content = write_consolidated_changelog(
                changelog_content, sections)

    # Write the changelog to file
    try:
        with open(output_path, "w", encoding="utf-8") as f:
            f.write(changelog_content)
        print(
            f"Successfully generated {output_path} with {len(version_tags)} version entries")
        return 0
    except Exception as e:
        print(f"Error writing changelog to {output_path}: {e}")
        return 1


def cmd_extract(args: Any) -> int:
    """Extract the content for a specific version."""
    changelog_path = args.output or os.path.join(
        get_repo_root(), "CHANGELOG.md")
    if extract_version_content(changelog_path, args.version, args.output_file):
        return 0
    return 1


# ---- Main Entry Point ----

def main() -> None:
    parser = argparse.ArgumentParser(
        description="Unified Changelog Management Script")
    subparsers = parser.add_subparsers(dest="command")

    # Generate command
    generate_parser = subparsers.add_parser(
        "generate", help="Generate a changelog for a specific range of commits")
    generate_parser.add_argument("--from-tag", help="Starting tag for commits")
    generate_parser.add_argument(
        "--to-tag", help="Ending reference (tag or commit)")
    generate_parser.add_argument(
        "--version", help="Version to add (e.g. '1.2.3')")
    generate_parser.add_argument("--output", help="Output file path")
    generate_parser.add_argument("--dry-run", action="store_true",
                                 help="Don't write to file, just print what would be added")
    generate_parser.add_argument("--no-consolidate", action="store_true",
                                 help="Disable automatic consolidation of duplicate entries")

    # Update command
    update_parser = subparsers.add_parser(
        "update", help="Update an existing changelog with a new version")
    update_parser.add_argument("--from-tag", help="Starting tag for commits")
    update_parser.add_argument(
        "--to-tag", help="Ending reference (tag or commit)")
    update_parser.add_argument("--version", required=True,
                               help="Version to add (e.g. '1.2.3')")
    update_parser.add_argument("--output", help="Output file path")
    update_parser.add_argument("--dry-run", action="store_true",
                               help="Don't write to file, just print what would be added")
    update_parser.add_argument("--no-consolidate", action="store_true",
                               help="Disable automatic consolidation of duplicate entries")

    # Full command
    full_parser = subparsers.add_parser(
        "full", help="Generate a full changelog from all version tags")
    full_parser.add_argument("--output", help="Output file path")
    full_parser.add_argument("--no-consolidate", action="store_true",
                             help="Disable automatic consolidation of duplicate entries")

    # Extract command
    extract_parser = subparsers.add_parser(
        "extract", help="Extract the content for a specific version")
    extract_parser.add_argument("--version", required=True,
                                help="Version to extract (e.g. '1.2.3')")
    extract_parser.add_argument("--output-file", required=True,
                                help="File to write extracted content to")

    args = parser.parse_args()

    if args.command == "generate":
        exit(cmd_generate(args))
    elif args.command == "update":
        exit(cmd_update(args))
    elif args.command == "full":
        exit(cmd_full(args))
    elif args.command == "extract":
        exit(cmd_extract(args))
    else:
        parser.print_help()
        exit(1)


if __name__ == "__main__":
    main()
