# Copyright (c) 2009-present, Homebrew contributors
# See the LICENSE file for extended copyright information.

class QuickLintJs < Formula
  desc "Find bugs in your JavaScript code"
  homepage "https://quick-lint-js.com/"
  url "https://c.quick-lint-js.com/releases/2.2.0/source/quick-lint-js-2.2.0.tar.gz"
  license "GPL-3.0-or-later"
  head "https://github.com/quick-lint/quick-lint-js.git", branch: "master"

  depends_on "cmake" => :build

  fails_with :clang do
    build 1100  # Xcode 11.3.1
    cause "Boost.JSON doesn't like Clang's std::string_view"
  end

  def install
    mkdir "build" do
      system "cmake", "..", *std_cmake_args,
                      "-DQUICK_LINT_JS_INSTALL_EMACS_DIR=share/emacs/site-lisp/quick-lint-js",
                      "-DQUICK_LINT_JS_INSTALL_VIM_NEOVIM_TAGS=ON"
      system "cmake", "--build", "."
      system "cmake", "--install", "."
    end
  end

  test do
    system "quick-lint-js", "--version"
  end
end
