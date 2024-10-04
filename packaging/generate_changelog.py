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
    with open(release_notes_file, 'r') as f:
        all_notes = f.read()

    current_date = datetime.now().strftime("%a %b %d %Y")
    packager_name = "Anar Manafov"
    packager_email = "Anar.Manafov@gmail.com"
    changelog_entry = ""

    for line in all_notes.splitlines():
        match = re.match(r'^## \[(.*)\] - (.*)$', line)
        if match:
            version, date = match.groups()
            if date == "NOT YET RELEASED":
                date = current_date
            else:
                date = convert_date(date)
            changelog_entry += f"* {date} {packager_name} <{packager_email}> - {version}-1\n"
        else:
            changelog_entry += f"{line}\n"

    with tempfile.NamedTemporaryFile(delete=False) as temp_changelog:
        temp_changelog.write(changelog_entry.encode())
        temp_changelog_path = temp_changelog.name

    with open(spec_file, 'r') as f:
        spec_content = f.read()

    with open(spec_file, 'w') as f:
        f.write(re.sub(r'(^%changelog)',
                f'\\1\n{changelog_entry}', spec_content, flags=re.MULTILINE))

    os.remove(temp_changelog_path)


if __name__ == "__main__":
    main(sys.argv[1], sys.argv[2])
