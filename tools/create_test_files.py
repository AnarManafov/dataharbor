import sys
import os
import random
import string


def generate_random_filename(length=8):
    """Generate a random filename with the given length."""
    letters = string.ascii_lowercase
    return ''.join(random.choice(letters) for i in range(length))


def create_files(num_files):
    """Create num_files with random names and small content."""
    for _ in range(num_files):
        filename = generate_random_filename() + '.txt'
        with open(filename, 'w') as f:
            f.write('This is a random file.\n')


if __name__ == "__main__":
    if len(sys.argv) != 2:
        print("Usage: python script.py <number_of_files>")
        sys.exit(1)

    try:
        num_files = int(sys.argv[1])
    except ValueError:
        print("Please provide a valid integer for the number of files.")
        sys.exit(1)

    create_files(num_files)
    print(f"{num_files} files created.")
