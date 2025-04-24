import sys
import re
import tempfile
import os
from datetime import datetime


def convert_date(input_date):
    try:
        return datetime.strptime(input_date, "%Y-%m-%d").strftime("%a %b %d %Y")
    except ValueError:
        return input_date


def main(spec_file, release_notes_file):
    if not os.path.exists(release_notes_file):
        print(f"Warning: Release notes file not found: {release_notes_file}")
        return

    with open(release_notes_file, 'r') as f:
        all_notes = f.read()

    current_date = datetime.now().strftime("%a %b %d %Y")
    packager_name = "Anar Manafov"
    packager_email = "Anar.Manafov@gmail.com"

    # Process the release notes
    version = None
    date = None
    entries = []
    version_entries = {}

    for line in all_notes.splitlines():
        # Detect version headers
        match = re.match(r'^## \[(.*)\] - (.*)$', line)
        if match:
            version, date = match.groups()
            if date == "NOT YET RELEASED":
                date = current_date
            else:
                date = convert_date(date)

            version_entries[version] = {
                'date': date,
                'entries': []
            }
            continue

        # Skip empty lines and section headers
        if not line.strip() or line.startswith('#'):
            continue

        # Add content lines to current version
        if version and line.strip().startswith('- '):
            # Remove leading "- " and whitespace
            clean_line = line.strip()[2:].strip()
            if clean_line:
                version_entries[version]['entries'].append(clean_line)

    # Format the changelog
    changelog_entries = []
    for ver in version_entries:
        entry_date = version_entries[ver]['date']
        changelog_entries.append(
            f"* {entry_date} {packager_name} <{packager_email}> - {ver}-1")

        for item in version_entries[ver]['entries']:
            changelog_entries.append(f"- {item}")

    changelog_content = "\n".join(changelog_entries)

    # If we have valid changelog content, update the spec file
    if changelog_content:
        with open(spec_file, 'r') as f:
            spec_content = f.read()

        with open(spec_file, 'w') as f:
            # Replace the %changelog section with our new content
            updated_content = re.sub(r'%changelog\s*$', f'%changelog\n{changelog_content}',
                                     spec_content, flags=re.MULTILINE)
            f.write(updated_content)

        print(f"Successfully updated changelog in {spec_file}")
    else:
        print(f"Warning: No changelog entries found in {release_notes_file}")


if __name__ == "__main__":
    if len(sys.argv) != 3:
        print("Usage: generate_changelog.py <spec_file> <release_notes_file>")
        sys.exit(1)
    main(sys.argv[1], sys.argv[2])
