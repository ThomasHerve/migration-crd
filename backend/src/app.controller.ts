import { AppService } from './app.service';
import { Entry } from './typeorm';
import { Body, Controller, Get, Post, Request } from '@nestjs/common';
import { CreateEntryDto } from 'src/dto/data.dto';

@Controller()
export class AppController {
  constructor(private readonly appService: AppService) {}

  @Get()
  async getHello(): Promise<Entry[]> {
    return await this.appService.getHello();
  }

  @Post('create')
  async createEntry(@Body() createEntryDto: CreateEntryDto, @Request() req) {
    try {
      return this.appService.createEntry(createEntryDto.title);
    } catch(e) {
      throw e;
    }
  }
}
