class Headroom < Formula
  desc "Intelligent context compression for AI agents"
  homepage "https://github.com/superops-team/headroom-go"
  version "0.7.0"

  if OS.mac?
    if Hardware::CPU.arm?
      url "https://github.com/superops-team/headroom-go/releases/download/v0.7.0/headroom-darwin-arm64"
      sha256 "REPLACE_WITH_ACTUAL_SHA256"
    else
      url "https://github.com/superops-team/headroom-go/releases/download/v0.7.0/headroom-darwin-amd64"
      sha256 "REPLACE_WITH_ACTUAL_SHA256"
    end
  elsif OS.linux?
    if Hardware::CPU.arm?
      url "https://github.com/superops-team/headroom-go/releases/download/v0.7.0/headroom-linux-arm64"
      sha256 "REPLACE_WITH_ACTUAL_SHA256"
    else
      url "https://github.com/superops-team/headroom-go/releases/download/v0.7.0/headroom-linux-amd64"
      sha256 "REPLACE_WITH_ACTUAL_SHA256"
    end
  end

  def install
    bin.install Dir["headroom-*"].first => "headroom"
  end

  test do
    assert_match "headroom-go", shell_output("#{bin}/headroom version")
  end
end
