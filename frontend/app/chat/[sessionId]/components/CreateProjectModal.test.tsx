import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { CreateProjectModal } from './CreateProjectModal';

describe('CreateProjectModal', () => {
  it('does not submit when Enter is used to confirm IME input', () => {
    const onSubmit = vi.fn().mockResolvedValue(undefined);

    render(
      <CreateProjectModal
        isOpen
        onClose={vi.fn()}
        onSubmit={onSubmit}
      />
    );

    const input = screen.getByLabelText('项目名称');

    fireEvent.change(input, { target: { value: 'nihao' } });
    fireEvent.keyDown(input, {
      key: 'Enter',
      isComposing: true,
      keyCode: 229,
    });

    expect(onSubmit).not.toHaveBeenCalled();
  });
});
