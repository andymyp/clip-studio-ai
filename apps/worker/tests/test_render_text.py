from pathlib import Path

from app.services.render import (
    _drawtext_file,
    _fit_title_font_size,
    _wrap_title,
    _write_overlay,
    _write_title_overlay,
)


def test_drawtext_file_keeps_apostrophes_and_url_characters_out_of_filter(
    tmp_path: Path,
) -> None:
    text = "Can you give love if you've never known it? https://youtube.com/watch?v=x"
    path = _write_overlay(tmp_path / "overlay.txt", text)
    filter_value = _drawtext_file(path, 125)

    assert path.read_text(encoding="utf-8").replace("\n", " ") == text
    assert "you've" not in filter_value
    assert "watch?v=x" not in filter_value
    assert "textfile=" in filter_value


def test_title_wrap_balances_large_centered_hook_across_three_lines() -> None:
    title = _wrap_title("This is the secret nobody tells you about success")

    lines = title.splitlines()
    assert 2 <= len(lines) <= 3
    assert " ".join(lines) == "This is the secret nobody tells you about success"
    assert max(map(len, lines)) - min(map(len, lines)) <= 8
    assert _fit_title_font_size(title) >= 80


def test_title_overlay_removes_markdown_without_changing_title_layout(
    tmp_path: Path,
) -> None:
    path = _write_title_overlay(
        tmp_path / "hook.txt", "**A Better Hook Changes Everything**"
    )

    assert path.read_text(encoding="utf-8") == "A Better Hook\nChanges Everything"
