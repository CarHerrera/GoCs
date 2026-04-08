import { Image } from 'react-konva';
import useImage from 'use-image';
import Konva from 'konva';
interface URLImageProps extends Omit<Konva.ImageConfig, 'image'> {
    src: string;
}
export function URLImage({ src,...rest}: URLImageProps) {
    // Hooks must be at the top level of the function
    const [img, status] = useImage(src, 'anonymous');
    if (status === 'loading') return null;

    // Now 'rest' contains width, height, x, y, etc.
    return <Image image={img} {...rest} />;
}