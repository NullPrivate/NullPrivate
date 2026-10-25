import React, { memo } from 'react';

type Props = {
    className?: string;
};

export const Logo = memo(({ className: _className }: Props) => {
    return (
        <svg xmlns="http://www.w3.org/2000/svg" width="160" height="40" viewBox="0 0 160 40">
            <image href="assets/favicon.png" x="0" y="0" width="40" height="40" />
            <text
                x="50"
                y="24"
                fill="#1a73e8"
                alignmentBaseline="middle"
                fontWeight="600"
                fontFamily="Arial, sans-serif"
                fontSize="24px"
                letterSpacing="0em"
                textAnchor="start"
            >
                宁屏
            </text>
        </svg>
    );
});

Logo.displayName = 'Logo';
