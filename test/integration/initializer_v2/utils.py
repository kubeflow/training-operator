import os


def verify_downloaded_files(dir_path, expected_files):
    """Verify downloaded files"""
    if expected_files:
        actual_files = set(os.listdir(dir_path))
        missing_files = set(expected_files) - actual_files
        assert not missing_files, f"Missing expected files: {missing_files}"
