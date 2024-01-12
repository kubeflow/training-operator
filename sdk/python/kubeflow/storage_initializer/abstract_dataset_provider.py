from abc import ABC, abstractmethod


class datasetProvider(ABC):
    @abstractmethod
    def load_config(self):
        pass

    @abstractmethod
    def download_dataset(self):
        pass
