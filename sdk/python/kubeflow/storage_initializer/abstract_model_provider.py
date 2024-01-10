from abc import ABC, abstractmethod


class modelProvider(ABC):
    @abstractmethod
    def load_config(self):
        pass

    @abstractmethod
    def download_model_and_tokenizer(self):
        pass
