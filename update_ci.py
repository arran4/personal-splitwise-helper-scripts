import re

with open('.github/workflows/ci.yml', 'r') as f:
    content = f.read()

# 1. Add top comment
header = """# Agent rules for generation:
# https://arran4.com/post/2026/006-Github-CI-and-Deploy/
# Built using this post as a reference/guide.
"""

if not content.startswith("# Agent"):
    content = header + content

# 2. Remove go-fmt-pr
# The go-fmt-pr job starts with "  go-fmt-pr:" and ends before "  autofix:"
content = re.sub(r'  go-fmt-pr:.*?  autofix:', '  autofix:', content, flags=re.DOTALL)

# 3. Remove publish-draft and promote-release
content = re.sub(r'  publish-draft:.*', '', content, flags=re.DOTALL)

with open('.github/workflows/ci.yml', 'w') as f:
    f.write(content.rstrip() + '\n')
