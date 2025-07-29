import { IsNotEmpty, MinLength } from "class-validator";

export class CreateEntryDto {
    @IsNotEmpty()
    @MinLength(3)
    title: string
}


