# /// script
# requires-python = ">=3.10"
# dependencies = ["go-to-wheel==0.2"]
# ///
"""Build subspace-tunnel wheels for all platforms: uv run scripts/build_wheels.py <version> [output_dir]"""

import os
import subprocess
import sys
import tempfile
from pathlib import Path

import go_to_wheel
from go_to_wheel import build_wheel

PLATFORMS = [
    ("linux", "amd64", "manylinux_2_17_x86_64"),
    ("linux", "arm64", "manylinux_2_17_aarch64"),
    ("linux", "amd64", "musllinux_1_2_x86_64"),
    ("linux", "arm64", "musllinux_1_2_aarch64"),
    ("darwin", "amd64", "macosx_12_0_x86_64"),
    ("darwin", "arm64", "macosx_12_0_arm64"),
    ("windows", "amd64", "win_amd64"),
    ("windows", "arm64", "win_arm64"),
]


def generate_entry_points(entry_point: str, import_name: str) -> str:
    return f"""[console_scripts]
{entry_point} = {import_name}:main
subspace-tunnel = {import_name}:main
"""


go_to_wheel.generate_entry_points = generate_entry_points


def main():
    version = sys.argv[1]
    output_dir = sys.argv[2] if len(sys.argv) > 2 else "dist"
    root = Path(__file__).resolve().parent.parent
    readme = (root / "README.md").read_text()
    os.makedirs(output_dir, exist_ok=True)

    with tempfile.TemporaryDirectory() as tmp:
        for goos, goarch, platform_tag in PLATFORMS:
            binary_path = os.path.join(tmp, f"subspace-{goos}-{goarch}")
            subprocess.run(
                ["go", "build", "-trimpath", "-ldflags=-s -w", "-o", binary_path, "./cmd/subspace"],
                cwd=root,
                env={**os.environ, "GOOS": goos, "GOARCH": goarch, "CGO_ENABLED": "0"},
                check=True,
            )
            wheel_path = build_wheel(
                binary_path=binary_path,
                output_dir=output_dir,
                name="subspace-tunnel",
                version=version,
                platform_tag=platform_tag,
                entry_point="subspace",
                is_windows=goos == "windows",
                description="Expose local HTTP services through a public relay server",
                author="Vladimir Malinovskii",
                author_email="galqiwi@gmail.com",
                url="https://github.com/galqiwi/subspace",
                readme_content=readme,
            )
            print(wheel_path)


if __name__ == "__main__":
    main()
