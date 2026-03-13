import codecs
import re

with open('data/models/treatments.go', 'r', encoding='utf-8') as f:
    text = f.read()

match = re.search(r'case .¿⁄±‚¿Â.:.*?(?s:.*?)(?=case constants\.K_PT:)', text)
if match:
    print(match.group(0))
else:
    print('Not found')
