# Use an official Python runtime as a parent image
FROM python:3.11

# Set the working directory in the container
WORKDIR /app

# Copy the requirements.txt file into the container
COPY requirements.txt /app/requirements.txt

# Install any needed packages specified in requirements.txt
RUN pip install --no-cache-dir torch==2.5.1
RUN pip install --no-cache-dir -r requirements.txt

# Copy the Python package and its source code into the container
COPY . /app

# Run storage.py when the container launches
ENTRYPOINT ["torchrun", "hf_llm_training.py"]
