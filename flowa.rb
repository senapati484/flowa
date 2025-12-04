# Homebrew Formula for Flowa
# To install: brew install --formula flowa.rb

class Flowa < Formula
  desc "A Pythonic, pipeline-first programming language designed for data processing and automation, emphasizing readability and ease of use."
  homepage "https://github.com/senapati484/flowa"
  url "https://github.com/senapati484/flowa/archive/refs/heads/main.tar.gz"
  version "0.1.2"
  license "MIT"

  depends_on "go" => :build

  def install
    system "go", "build", "-o", "flowa", "./cmd/flowa"
    bin.install "flowa"
    
    # Install examples
    pkgshare.install "examples"
  end

  test do
    (testpath/"test.flowa").write <<~EOS
      func add(x, y){
          return x + y
      }
      
      result = add(5, 10)
    EOS
    system "#{bin}/flowa", "test.flowa"
  end
end
