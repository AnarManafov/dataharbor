#!/usr/bin/env python3
"""
Release Notes Generator Script

Generates release notes by leveraging the CHANGELOG.md content, but with 
more user-friendly formatting and additional information relevant for releases.

Usage:
  python generate_release_notes.py [--version VERSION] [--from-tag TAG] [--to-tag TAG] [--output-file FILE]

Options:
  --version      The version to generate release notes for (required)
  --from-tag     Get commits starting from this tag (default: previous tag)
  --to-tag       Get commits up to this tag (default: v{VERSION})
  --output-file  Where to write the release notes (default: RELEASE_NOTES.md)
"""

import argparse
import os
import re
import subprocess
from datetime import date
import sys
import json
from typing import Dict, List, Optional, Tuple

# Import functions from changelog.py
try:
    from changelog import (
        run_git_command,
        get_repo_root,
        get_latest_tag,
        extract_ticket_reference,
        get_commits,
        group_commits_by_type,
        generate_version_section
    )
except ImportError:
    print("Error: Could not import from changelog.py. Make sure it's in the same directory.")
    sys.exit(1)


def parse_args():
    """Parse command line arguments."""
    parser = argparse.ArgumentParser(
        description="Generate release notes from changelog and git history")
    parser.add_argument("--version", required=True,
                        help="Version to generate release notes for")
    parser.add_argument(
        "--from-tag", help="Get commits starting from this tag (default: previous tag)")
    parser.add_argument(
        "--to-tag", help="Get commits up to this tag (default: v{VERSION})")
    parser.add_argument("--output-file", default="RELEASE_NOTES.md",
                        help="File to write release notes to (default: RELEASE_NOTES.md)")

    return parser.parse_args()


def get_version_tags() -> List[str]:
    """Get all version tags in the repository sorted by version."""
    try:
        all_tags = run_git_command(["tag", "--sort=v:refname"]).split("\n")
        return [tag for tag in all_tags if re.match(r"^v\d+\.\d+\.\d+$", tag)]
    except Exception as e:
        print(f"Error getting version tags: {e}")
        return []


def find_prev_and_current_tags(version: str) -> Tuple[Optional[str], Optional[str]]:
    """Find the previous and current tag for a given version."""
    version_tag = f"v{version}"
    version_tags = get_version_tags()

    if not version_tags:
        print("No version tags found in the repository")
        return None, None

    # Find current tag
    if version_tag in version_tags:
        current_tag = version_tag
        current_idx = version_tags.index(current_tag)

        # Find previous tag
        prev_tag = None
        if current_idx > 0:
            prev_tag = version_tags[current_idx - 1]
        else:
            # First tag - use first commit
            prev_tag = run_git_command(["rev-list", "--max-parents=0", "HEAD"])

        return prev_tag, current_tag
    else:
        # Tag doesn't exist yet - this is likely during CI processing of a trigger tag
        # Find the most recent version tag as the previous tag
        if version_tags:
            prev_tag = version_tags[-1]  # Most recent tag
            print(
                f"Note: Target tag v{version} not found yet (likely during CI processing). Using {prev_tag} as previous tag.", file=sys.stderr)
        else:
            # No version tags at all - use first commit
            prev_tag = run_git_command(["rev-list", "--max-parents=0", "HEAD"])
            print(
                f"Note: No version tags found. Using first commit as previous reference.", file=sys.stderr)

        return prev_tag, None


def get_commits_by_type_with_tickets(from_tag: str, to_tag: str) -> Dict[str, Dict[str, List[str]]]:
    """Get commits by conventional commit type between two Git references with ticket references."""
    print(f"Getting commits between {from_tag}..{to_tag}", file=sys.stderr)

    # Use the `get_commits` function from changelog.py to fetch structured commit data
    commits = get_commits(from_tag, to_tag)

    # Use the `group_commits_by_type` function from changelog.py to group commits by type
    commits_by_type = group_commits_by_type(commits)

    return commits_by_type


def get_commit_summary(from_tag: str, to_tag: str) -> str:
    """Get a summary of commits between two tags for release notes."""
    commits_by_type = get_commits_by_type_with_tickets(from_tag, to_tag)

    if not commits_by_type:
        return "> No changes found for this version."

    content = []

    for commit_type_data in commits_by_type.values():
        if commit_type_data["items"]:
            content.append(commit_type_data["title"])
            for commit in commit_type_data["items"]:
                content.append(commit)
            content.append("")

    return "\n".join(content).strip()


def generate_release_notes(version: str, from_tag: str = None, to_tag: str = None) -> str:
    """Generate release notes content."""
    repo_root = get_repo_root()
    today = date.today().isoformat()

    # Determine from_tag and to_tag if not specified
    if not from_tag or not to_tag:
        prev_tag, curr_tag = find_prev_and_current_tags(version)

        if not from_tag:
            from_tag = prev_tag

        if not to_tag:
            # If curr_tag doesn't exist (e.g., during CI processing), use HEAD
            # which represents the current state that will become the release
            to_tag = curr_tag if curr_tag else "HEAD"

    if not from_tag:
        print("Error: Could not determine from_tag", file=sys.stderr)
        sys.exit(1)

    if not to_tag:
        print("Error: Could not determine to_tag", file=sys.stderr)
        sys.exit(1)

    # Print information to stderr so it doesn't interfere with output redirection
    commit_count = run_git_command(
        ["rev-list", "--count", f"{from_tag}..{to_tag}"]
    ).strip()
    print(
        f"Generating changelog from {from_tag} to {to_tag}...", file=sys.stderr)
    print(f"Found {commit_count} commits.", file=sys.stderr)

    # Generate commit summary from the specific range
    commit_summary = get_commit_summary(from_tag, to_tag)

    if not commit_summary or commit_summary == "> No changes found for this version.":
        changelog_content = "> No changes found for this version."
    else:
        changelog_content = commit_summary

    # Format release notes - without technical markers, version info section only
    # Removed the GitHub Actions autogenerated footer to keep notes clean.
    release_notes = f"""## v{version} ({today})

Changelog:
{changelog_content}
"""

    # Store the raw changelog content separately for CI tools that might need it
    raw_changelog = changelog_content

    return release_notes, raw_changelog


def update_release_notes(release_notes_content: str, output_file: str) -> None:
    """Update the RELEASE_NOTES.md file with new content."""
    repo_root = get_repo_root()
    release_notes_path = os.path.join(repo_root, output_file)

    try:
        # Simply write the new content, replacing any existing file
        # This ensures only the latest version is included
        with open(release_notes_path, "w", encoding="utf-8") as f:
            f.write(f"# Release Notes\n\n{release_notes_content}")

        print(f"Successfully updated {output_file}", file=sys.stderr)

    except Exception as e:
        print(f"Error updating release notes: {e}")
        sys.exit(1)


def save_raw_changelog(raw_content: str, version: str) -> None:
    """Save the raw changelog content to a separate file for CI tools."""
    repo_root = get_repo_root()
    raw_path = os.path.join(repo_root, f"CHANGELOG_RAW_{version}.txt")

    try:
        with open(raw_path, "w", encoding="utf-8") as f:
            f.write(f"GENERATED_CHANGELOG<<EOF\n{raw_content}\nEOF\n")
        print(
            f"Saved raw changelog to {raw_path} for CI tools", file=sys.stderr)
    except Exception as e:
        print(f"Error saving raw changelog: {e}")


if __name__ == "__main__":
    args = parse_args()

    # Generate release notes content
    release_notes_content, raw_changelog = generate_release_notes(
        args.version,
        from_tag=args.from_tag,
        to_tag=args.to_tag
    )

    # For CI workflows that use shell redirection, output the raw changelog content to stdout
    # The workflow expects just the changelog content, not the full release notes format
    print(raw_changelog)

    # Also save to file for local use and CI tools
    save_raw_changelog(raw_changelog, args.version)
