"""Various utilities used by scripts."""


def run(command, cwd=None):
  logging.info("Running: %s", " ".join(command))
  subprocess.check_call(command, cwd=cwd)


def run_and_output(command, cwd=None):
  logging.info("Running: %s", " ".join(command))
  # The output won't be available until the command completes.
  # So prefer using run if we don't need to return the output.
  output = subprocess.check_output(command, cwd=cwd).decode("utf-8")
  print(output)
  return output
