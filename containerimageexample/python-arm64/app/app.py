import sys
import time


def handler(event, context):
    time.sleep(2)
    return 'Hello from AWS Lambda using Python' + sys.version + '!'
