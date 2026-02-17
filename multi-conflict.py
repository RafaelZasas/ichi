#!/usr/bin/env python3
"""
Multi-file conflict test
"""

class ConfigManager:
    def __init__(self):
        self.version = "2.0"
        self.features = ["feature-x", "feature-y"]

    def get_config(self):
        return {
            "version": self.version,
            "features": self.features,
            "mode": "feature-mode"
        }

if __name__ == "__main__":
    manager = ConfigManager()
    print(manager.get_config())
