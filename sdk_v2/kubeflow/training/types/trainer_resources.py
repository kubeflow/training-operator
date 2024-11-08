# Representation for the Training Runtime.

from dataclasses import dataclass


@dataclass
class Runtime:
    name: str
    stage: str
